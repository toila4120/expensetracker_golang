# GOOGLE OAUTH - FE INTEGRATION GUIDE

**Version:** 2.0  
**Last Updated:** 2026-06-30  
**Cập nhật:** Thêm aud validation, fix password constraint

---

## Overview

Đăng nhập bằng Google Account. Backend sử dụng Google ID Token để xác thực.

**Flow:**
```
Frontend → Google Sign-In → Lấy ID Token → Gửi POST /auth/google → Nhận JWT tokens
```

---

## Setup Google Cloud Console

### Bước 1: Tạo OAuth 2.0 Client ID

1. Vào [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Chọn hoặc tạo Project
3. Vào **APIs & Services > Credentials**
4. Click **Create Credentials > OAuth client ID**
5. Chọn **Web application**
6. Đặt tên (ví dụ: "ExpenseTracker Web")
7. Thêm **Authorized JavaScript origins**:
   - `http://localhost:3000` (Flutter web dev)
   - `http://localhost:8080` (nếu chạy cùng backend)
   - `https://yourdomain.com` (production)
8. Copy **Client ID** và **Client Secret**

### Bước 2: Cấu hình Backend

```env
# .env
GOOGLE_CLIENT_ID=123456789-xxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxx
JWT_SECRET=your-random-secret-key
```

---

## API Endpoint

```
POST /auth/google
```

Không cần Authorization header - đây là public endpoint.

### Request Body

**Option 1: Using ID Token (recommended)**
```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Option 2: Using Access Token**
```json
{
  "access_token": "ya29.a0AfH6SMB..."
}
```

### Response

**201 Created - New user registered**
```json
{
  "message": "Đăng ký thành công bằng Google",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "is_new_user": true
}
```

**200 OK - Existing user logged in**
```json
{
  "message": "Đăng nhập thành công",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "is_new_user": false
}
```

**200 OK - Account linked**
```json
{
  "message": "Đăng nhập thành công và đã liên kết tài khoản Google",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "is_new_user": false,
  "is_linked": true
}
```

### Error Responses

| Status | Error | Meaning |
|--------|-------|---------|
| 400 | "Dữ liệu đầu vào không hợp lệ" | Thiếu id_token hoặc access_token |
| 401 | "Token Google không hợp lệ" | Token sai, hết hạn, hoặc aud mismatch |
| 401 | "Email Google chưa được xác thực" | Email chưa verified |
| 500 | "Không thể tạo tài khoản" | Lỗi DB |

---

## Frontend Integration

### Option 1: Google Identity Services (Recommended)

```html
<!-- Add to <head> -->
<script src="https://accounts.google.com/gsi/client" async defer></script>
```

```javascript
// Initialize Google Sign-In
function initializeGoogleSignIn() {
  google.accounts.id.initialize({
    client_id: 'YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com',
    callback: handleGoogleResponse
  });
  
  // Render button
  google.accounts.id.renderButton(
    document.getElementById('google-signin-btn'),
    { theme: 'outline', size: 'large', text: 'signin_with' }
  );
}

// Handle response from Google
async function handleGoogleResponse(response) {
  // response.credential is the ID token
  const idToken = response.credential;
  
  try {
    const result = await fetch('/auth/google', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id_token: idToken })
    });
    
    const data = await result.json();
    
    if (result.ok) {
      // Store tokens
      localStorage.setItem('access_token', data.access_token);
      localStorage.setItem('refresh_token', data.refresh_token);
      
      // Check if new user (for onboarding flow)
      if (data.is_new_user) {
        // Navigate to onboarding
        window.location.href = '/onboarding';
      } else {
        // Navigate to home
        window.location.href = '/';
      }
    } else {
      showError(data.error);
    }
  } catch (error) {
    console.error('Google login error:', error);
    showError('Lỗi kết nối server');
  }
}
```

### Option 2: Flutter Web

```dart
import 'package:google_sign_in/google_sign_in.dart';
import 'dart:html' as html;

final GoogleSignIn _googleSignIn = GoogleSignIn(
  clientId: 'YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com',
  scopes: ['email', 'profile'],
);

Future<void> signInWithGoogle() async {
  try {
    final GoogleSignInAccount? account = await _googleSignIn.signIn();
    if (account == null) return; // User cancelled
    
    final GoogleSignInAuthentication auth = await account.authentication;
    final String? idToken = auth.idToken;
    
    if (idToken == null) {
      showError('Không thể lấy ID token');
      return;
    }
    
    // Send to backend
    final response = await http.post(
      Uri.parse('$BASE_URL/auth/google'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'id_token': idToken}),
    );
    
    final data = jsonDecode(response.body);
    
    if (response.statusCode == 200 || response.statusCode == 201) {
      // Store tokens
      await storage.write(key: 'access_token', value: data['access_token']);
      await storage.write(key: 'refresh_token', value: data['refresh_token']);
      
      // Navigate based on is_new_user
      if (data['is_new_user'] == true) {
        Navigator.pushReplacementNamed(context, '/onboarding');
      } else {
        Navigator.pushReplacementNamed(context, '/home');
      }
    } else {
      showError(data['error']);
    }
  } catch (error) {
    showError('Lỗi kết nối: $error');
  }
}
```

---

## Business Rules

### Account Linking Logic

```
1. Email chưa tồn tại trong hệ thống
   → Tạo tài khoản mới với provider="google"
   → Return is_new_user: true

2. Email đã tồn tại với provider="local" (đăng ký bằng password)
   → Ghép tài khoản Google vào tài khoản cũ
   → Cập nhật provider="google", provider_id=google_id
   → Return is_linked: true

3. Email đã tồn tại với provider="google" (đã link rồi)
   → Đăng nhập bình thường
   → Return is_new_user: false
```

### User Model

```typescript
interface User {
  id: number;
  username: string;
  email: string;
  provider: "local" | "google";
  provider_id?: string;  // Google ID
  created_at: string;
}
```

---

## Important Notes

1. **Password field**: Google users không có password. Nếu `provider === "google"`, ẩn nút "Đổi mật khẩu"

2. **Account linking**: Nếu user đăng ký email/password rồi sau đó đăng nhập Google cùng email, hệ thống tự động ghép tài khoản

3. **JWT tokens**: Sau Google login, response trả về `access_token` và `refresh_token` - giống hệt đăng nhập thường

4. **Client ID phải khớp**: Dùng cùng `GOOGLE_CLIENT_ID` giữa frontend và backend `.env`

5. **Aud validation**: Backend kiểm tra `aud` claim trong ID token phải khớp với `GOOGLE_CLIENT_ID` - tránh accepting token từ app khác

---

## Testing

1. **Development**: 
   - Thêm `http://localhost:3000` vào Authorized JavaScript origins trong Google Cloud Console
   - Dùng Google test account hoặc account thật

2. **Production**: 
   - Thêm domain production vào Authorized JavaScript origins
   - Cập nhật `GOOGLE_CLIENT_ID` trong `.env`

3. **Common Issues**:
   - `token.google.com` error: Client ID không đúng hoặc chưa thêm origins
   - "Email chưa verified": Cần verify email trong Google account
   - CORS error: Kiểm tra origins đã đúng chưa

---

## Error Handling

```typescript
async function signInWithGoogle(idToken: string) {
  try {
    const response = await fetch('/auth/google', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id_token: idToken })
    });
    
    const data = await response.json();
    
    switch (response.status) {
      case 200:
        // Existing user logged in
        handleLoginSuccess(data);
        break;
      case 201:
        // New user registered
        handleNewUser(data);
        break;
      case 400:
        // Missing token - check implementation
        showError('Thiếu thông tin xác thực');
        break;
      case 401:
        // Invalid token - prompt re-login
        if (data.error.includes('aud')) {
          showError('Client ID không hợp lệ');
        } else {
          showError('Token không hợp lệ, vui lòng đăng nhập lại');
        }
        break;
      default:
        // Server error
        showError('Lỗi server, vui lòng thử lại sau');
    }
  } catch (error) {
    // Network error
    showError('Lỗi kết nối mạng');
  }
}
```
