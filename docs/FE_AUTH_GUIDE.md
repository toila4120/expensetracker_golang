# AUTH API - FE INTEGRATION GUIDE

**Version:** 1.0  
**Last Updated:** 2026-06-30  
**Base URL:** `http://localhost:8080`

---

## MỤC LỤC

1. [Overview](#1-overview)
2. [TypeScript Types](#2-typescript-types)
3. [API Reference](#3-api-reference)
4. [Token Management](#4-token-management)
5. [Integration Examples](#5-integration-examples)

---

## 1. Overview

Tất cả auth endpoints đều là **public** (không cần Authorization header).

```
POST   /auth/register    → Đăng ký tài khoản mới
POST   /auth/login       → Đăng nhập bằng email/password
POST   /auth/refresh     → Làm mới access token
POST   /auth/google      → Đăng nhập bằng Google
```

---

## 2. TypeScript Types

### Request Types

```typescript
// ==================== REGISTER ====================
interface RegisterRequest {
  email: string;      // required, valid email
  password: string;   // required, minimum 6 characters
}

// ==================== LOGIN ====================
interface LoginRequest {
  email: string;      // required, valid email
  password: string;   // required
}

// ==================== REFRESH TOKEN ====================
interface RefreshTokenRequest {
  refresh_token: string;  // required
}

// ==================== GOOGLE LOGIN ====================
interface GoogleLoginRequest {
  id_token?: string;       // Google ID token (preferred)
  access_token?: string;   // Google access token (alternative)
}
```

### Response Types

```typescript
// ==================== REGISTER RESPONSE ====================
interface RegisterResponse {
  message: string;  // "Đăng ký thành công"
}

// ==================== LOGIN RESPONSE ====================
interface LoginResponse {
  message: string;          // "Đăng nhập thành công"
  access_token: string;     // JWT access token (15 phút)
  refresh_token: string;    // JWT refresh token (7 ngày)
}

// ==================== REFRESH RESPONSE ====================
interface RefreshResponse {
  message: string;          // "Làm mới token thành công"
  access_token: string;     // New access token
}

// ==================== GOOGLE LOGIN RESPONSE ====================
interface GoogleLoginResponse {
  message: string;          // "Đăng nhập thành công"
  access_token: string;
  refresh_token: string;
  is_new_user?: boolean;    // true nếu tạo tài khoản mới
  is_linked?: boolean;      // true nếu ghép tài khoản cũ
}

// ==================== USER MODEL ====================
interface User {
  id: number;
  username: string;
  email: string;
  provider: "local" | "google";
  provider_id?: string;     // Google ID (nếu login bằng Google)
  created_at: string;
}

// ==================== ERROR RESPONSE ====================
interface ErrorResponse {
  error: string;            // Mô tả lỗi
  details?: string;         // Chi tiết lỗi (validation)
}
```

---

## 3. API Reference

### 3.1 POST `/auth/register` - Đăng Ký

```typescript
const register = async (data: RegisterRequest): Promise<RegisterResponse> => {
  const response = await fetch(`${BASE_URL}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  });
  
  const result = await response.json();
  
  if (!response.ok) {
    throw new Error(result.error);
  }
  
  return result;
};
```

**Request:**
```json
{
  "email": "user@example.com",
  "password": "matkhau123"
}
```

**Response 201:**
```json
{
  "message": "Đăng ký thành công"
}
```

**Errors:**
| Status | Error | Meaning |
|--------|-------|---------|
| 400 | "Dữ liệu đầu vào không hợp lệ" | Email hoặc password không hợp lệ |
| 409 | "Email đã tồn tại" | Email đã được đăng ký |
| 500 | "Không thể tạo tài khoản" | Lỗi DB |

---

### 3.2 POST `/auth/login` - Đăng Nhập

```typescript
const login = async (data: LoginRequest): Promise<LoginResponse> => {
  const response = await fetch(`${BASE_URL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  });
  
  const result = await response.json();
  
  if (!response.ok) {
    throw new Error(result.error);
  }
  
  return result;
};
```

**Request:**
```json
{
  "email": "user@example.com",
  "password": "matkhau123"
}
```

**Response 200:**
```json
{
  "message": "Đăng nhập thành công",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Errors:**
| Status | Error | Meaning |
|--------|-------|---------|
| 400 | "Dữ liệu đầu vào không hợp lệ" | Thiếu email hoặc password |
| 401 | "Email hoặc mật khẩu không đúng" | Sai email hoặc password |

---

### 3.3 POST `/auth/refresh` - Làm Mới Token

```typescript
const refreshToken = async (token: string): Promise<RefreshResponse> => {
  const response = await fetch(`${BASE_URL}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: token })
  });
  
  const result = await response.json();
  
  if (!response.ok) {
    throw new Error(result.error);
  }
  
  return result;
};
```

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response 200:**
```json
{
  "message": "Làm mới token thành công",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Errors:**
| Status | Error | Meaning |
|--------|-------|---------|
| 400 | "Vui lòng cung cấp refresh token" | Thiếu refresh_token |
| 401 | "Refresh token không hợp lệ hoặc đã hết hạn" | Token sai hoặc expired |
| 401 | "Người dùng không tồn tại hoặc đã bị xóa" | User đã bị xóa |

---

### 3.4 POST `/auth/google` - Đăng Nhập Google

Xem chi tiết tại [GOOGLE_OAUTH_FOR_FRONTEND.md](./GOOGLE_OAUTH_FOR_FRONTEND.md)

**Request:**
```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response 201 (new user):**
```json
{
  "message": "Đăng ký thành công bằng Google",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "is_new_user": true
}
```

**Response 200 (existing user):**
```json
{
  "message": "Đăng nhập thành công",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "is_new_user": false
}
```

**Response 200 (linked account):**
```json
{
  "message": "Đăng nhập thành công và đã liên kết tài khoản Google",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "is_new_user": false,
  "is_linked": true
}
```

---

## 4. Token Management

### Token Types

| Token | Lifetime | Purpose |
|-------|----------|---------|
| Access Token | 15 phút | Authenticate API requests |
| Refresh Token | 7 ngày | Get new access token |

### Token Flow

```
Login → Get access_token + refresh_token
  ↓
Use access_token in Authorization header
  ↓
When expired (401) → Call /auth/refresh with refresh_token
  ↓
Get new access_token → Continue
```

### Using Access Token

```typescript
// Header format
Authorization: Bearer <access_token>

// Example
const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${localStorage.getItem('access_token')}`
};
```

### Token Storage

```typescript
// localStorage (web)
localStorage.setItem('access_token', token);
localStorage.setItem('refresh_token', token);

// Flutter Secure Storage (mobile)
await storage.write(key: 'access_token', value: token);
await storage.write(key: 'refresh_token', value: token);
```

---

## 5. Integration Examples

### 5.1 Auth Service (TypeScript)

```typescript
const BASE_URL = 'http://localhost:8080';

class AuthService {
  // Store tokens after login
  static setTokens(accessToken: string, refreshToken: string) {
    localStorage.setItem('access_token', accessToken);
    localStorage.setItem('refresh_token', refreshToken);
  }

  // Clear tokens on logout
  static clearTokens() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
  }

  // Get stored access token
  static getAccessToken(): string | null {
    return localStorage.getItem('access_token');
  }

  // Check if user is logged in
  static isAuthenticated(): boolean {
    return !!this.getAccessToken();
  }

  // Register
  static async register(email: string, password: string) {
    const response = await fetch(`${BASE_URL}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });

    const data = await response.json();
    if (!response.ok) throw new Error(data.error);
    return data;
  }

  // Login
  static async login(email: string, password: string) {
    const response = await fetch(`${BASE_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });

    const data = await response.json();
    if (!response.ok) throw new Error(data.error);

    this.setTokens(data.access_token, data.refresh_token);
    return data;
  }

  // Login with Google
  static async loginWithGoogle(idToken: string) {
    const response = await fetch(`${BASE_URL}/auth/google`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id_token: idToken })
    });

    const data = await response.json();
    if (!response.ok) throw new Error(data.error);

    this.setTokens(data.access_token, data.refresh_token);
    return data;
  }

  // Refresh token
  static async refreshAccessToken() {
    const refreshToken = localStorage.getItem('refresh_token');
    if (!refreshToken) throw new Error('No refresh token');

    const response = await fetch(`${BASE_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken })
    });

    const data = await response.json();
    if (!response.ok) {
      this.clearTokens();
      throw new Error(data.error);
    }

    localStorage.setItem('access_token', data.access_token);
    return data.access_token;
  }

  // Logout
  static logout() {
    this.clearTokens();
    window.location.href = '/login';
  }
}

export default AuthService;
```

### 5.2 Protected API Call with Auto-Refresh

```typescript
async function apiCall(url: string, options: RequestInit = {}) {
  const token = AuthService.getAccessToken();
  
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
    ...options.headers
  };

  let response = await fetch(`${BASE_URL}${url}`, { ...options, headers });

  // If 401, try refresh token
  if (response.status === 401) {
    try {
      await AuthService.refreshAccessToken();
      headers['Authorization'] = `Bearer ${AuthService.getAccessToken()}`;
      response = await fetch(`${BASE_URL}${url}`, { ...options, headers });
    } catch (error) {
      // Refresh failed, redirect to login
      AuthService.logout();
      throw new Error('Session expired');
    }
  }

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error);
  }

  return response.json();
}

// Usage
const transactions = await apiCall('/api/transactions');
const profile = await apiCall('/api/profile');
```

### 5.3 Login Screen (Flutter)

```dart
class LoginScreen extends StatefulWidget {
  @override
  _LoginScreenState createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _isLoading = false;

  Future<void> _login() async {
    setState(() => _isLoading = true);
    
    try {
      final response = await http.post(
        Uri.parse('$BASE_URL/auth/login'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({
          'email': _emailController.text,
          'password': _passwordController.text,
        }),
      );

      final data = jsonDecode(response.body);

      if (response.statusCode == 200) {
        // Store tokens
        await storage.write(key: 'access_token', value: data['access_token']);
        await storage.write(key: 'refresh_token', value: data['refresh_token']);
        
        // Navigate to home
        Navigator.pushReplacementNamed(context, '/home');
      } else {
        _showError(data['error']);
      }
    } catch (e) {
      _showError('Lỗi kết nối');
    } finally {
      setState(() => _isLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Padding(
        padding: EdgeInsets.all(20),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            TextField(
              controller: _emailController,
              decoration: InputDecoration(labelText: 'Email'),
              keyboardType: TextInputType.emailAddress,
            ),
            SizedBox(height: 16),
            TextField(
              controller: _passwordController,
              decoration: InputDecoration(labelText: 'Mật khẩu'),
              obscureText: true,
            ),
            SizedBox(height: 24),
            ElevatedButton(
              onPressed: _isLoading ? null : _login,
              child: _isLoading 
                ? CircularProgressIndicator()
                : Text('Đăng nhập'),
            ),
            SizedBox(height: 16),
            ElevatedButton.icon(
              onPressed: () => _signInWithGoogle(),
              icon: Icon(Icons.g_mobiledata),
              label: Text('Đăng nhập bằng Google'),
            ),
          ],
        ),
      ),
    );
  }
}
```

### 5.4 Auth Guard (Route Protection)

```typescript
// React Router example
import { Navigate } from 'react-router-dom';

function ProtectedRoute({ children }) {
  if (!AuthService.isAuthenticated()) {
    return <Navigate to="/login" replace />;
  }
  return children;
}

// Usage
<Route path="/dashboard" element={
  <ProtectedRoute>
    <Dashboard />
  </ProtectedRoute>
} />
```

---

## QUICK REFERENCE CARD

```
┌─────────────────────────────────────────────────────────────────┐
│                    AUTH API QUICK REFERENCE                       │
├─────────────────────────────────────────────────────────────────┤
│  BASE: http://localhost:8080                                    │
│  AUTH: Không cần header cho auth endpoints                      │
├─────────────────────────────────────────────────────────────────┤
│  AUTH                                                            │
│  POST   /auth/register        → Register (email+password)       │
│  POST   /auth/login           → Login (email+password)          │
│  POST   /auth/refresh         → Refresh access token            │
│  POST   /auth/google          → Login with Google               │
├─────────────────────────────────────────────────────────────────┤
│  TOKEN LIFETIME                                                  │
│  Access Token: 15 minutes                                        │
│  Refresh Token: 7 days                                           │
├─────────────────────────────────────────────────────────────────┤
│  PROTECTED API CALL                                              │
│  Header: Authorization: Bearer <access_token>                   │
│  On 401: Call /auth/refresh with refresh_token                  │
└─────────────────────────────────────────────────────────────────┘
```
