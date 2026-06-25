# NOTIFICATION SERVICE - HƯỚNG DẪN SETUP & TÍCH HỢP

**Cập nhật:** 2026-06-25

---

## PHẦN 1: BACKEND SETUP

### 1.1 Cài đặt dependency

```bash
go get firebase.google.com/go/v4
go get gorm.io/datatypes
```

### 1.2 Thêm env variables vào `.env`

```env
# ========================
# FIREBASE (tùy chọn)
# Nếu không config thì FCM push bị disabled, hệ thống vẫn chạy bình thường
# ========================
FIREBASE_PROJECT_ID=your-firebase-project-id
FIREBASE_CREDENTIALS=./firebase-service-account.json

# ========================
# EMAIL SMTP (tùy chọn)
# Nếu không config thì email bị disabled, hệ thống vẫn chạy bình thường
# ========================
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
SMTP_FROM=ExpenseTracker <no-reply@expensetracker.com>
```

### 1.3 Firebase Setup

#### Bước 1: Tạo Firebase Project
1. Vào https://console.firebase.google.com
2. Click **"Create a project"**
3. Nhập tên project (VD: `expensetracker`)
4. Bật Google Analytics (tùy chọn) → Create

#### Bước 2: Thêm Android App
1. Trong Firebase Console → Click icon **Android**
2. Nhập Android package name: `com.example.expensetracker`
3. Nhập App nickname: `Expense Tracker`
4. Click **"Register app"**
5. Download `google-services.json` → đặt vào `android/app/` trong Flutter project

#### Bước 3: Tạo Service Account Key (cho Go Backend)
1. Firebase Console → ⚙️ **Project Settings** → **Service accounts**
2. Click **"Generate new private key"**
3. Lưu file JSON (VD: `firebase-service-account.json`)
4. Đặt file vào **root directory** của Go backend
5. **QUAN TRỌNG:** Thêm file này vào `.gitignore`:

```gitignore
.env
firebase-service-account.json
```

#### Bước 4: Cấu hình Flutter với Firebase
```yaml
# android/app/build.gradle
dependencies {
    implementation platform('com.google.firebase:firebase-bom:33.7.0')
    implementation 'com.google.firebase:firebase-messaging'
}
```

```yaml
# pubspec.yaml
dependencies:
  firebase_core: ^3.8.1
  firebase_messaging: ^15.2.1
```

### 1.4 Gmail SMTP Setup (nếu dùng Gmail)

1. Vào tài khoản Google → **Security** → **2-Step Verification** → Bật
2. Vào https://myaccount.google.com/apppasswords
3. Tạo app password mới (VD: `ExpenseTracker`)
4. Copy password 16 chữ cái → đặt vào `SMTP_PASS`

**Lưu ý:** Không dùng password chính của Gmail, phải dùng App Password.

### 1.5 Run Backend

```bash
# Install dependencies
go mod tidy

# Run
go run main.go
```

Server sẽ tự động:
- AutoMigrate bảng `notifications` và `fcm_tokens`
- Khởi tạo FCM Service (nếu có config)
- Khởi tạo Email Service (nếu có config)

---

## PHẦN 2: API REFERENCE

### 2.1 Đăng ký FCM Token

```
POST /api/fcm-token
Authorization: Bearer <token>

{
  "token": "fcm-device-token",
  "platform": "android"  // hoặc "ios"
}

Response 201:
{
  "message": "Đã đăng ký FCM token thành công"
}
```

### 2.2 Xóa FCM Token (khi logout)

```
DELETE /api/fcm-token
Authorization: Bearer <token>

{
  "token": "fcm-device-token"
}

Response 200:
{
  "message": "Đã xóa FCM token"
}
```

### 2.3 Lấy danh sách Notifications

```
GET /api/notifications?page=1&limit=20
Authorization: Bearer <token>

Response 200:
{
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "type": "budget_warning",
      "title": "Cảnh báo ngân sách",
      "message": "Chi tiêu danh mục Ăn uống đã vượt 80% ngân sách",
      "is_read": false,
      "metadata": null,
      "created_at": "2026-06-25T10:30:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "limit": 20
}
```

### 2.4 Lấy số Notifications chưa đọc

```
GET /api/notifications/unread-count
Authorization: Bearer <token>

Response 200:
{
  "count": 5
}
```

### 2.5 Đánh dấu đã đọc (1 notification)

```
PUT /api/notifications/:id/read
Authorization: Bearer <token>

Response 200:
{
  "message": "Đã đánh dấu đã đọc"
}
```

### 2.6 Đánh dấu tất cả đã đọc

```
PUT /api/notifications/read-all
Authorization: Bearer <token>

Response 200:
{
  "message": "Đã đánh dấu tất cả đã đọc"
}
```

---

## PHẦN 3: FLUTTER INTEGRATION

### 3.1 Thêm dependencies

```yaml
# pubspec.yaml
dependencies:
  firebase_core: ^3.8.1
  firebase_messaging: ^15.2.1
  flutter_local_notifications: ^18.0.1
  http: ^1.2.0
```

### 3.2 Firebase Initialization

```dart
// lib/main.dart
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'services/notification_service.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await Firebase.initializeApp();
  
  // Initialize notification service
  await NotificationService().initialize();
  
  runApp(MyApp());
}
```

### 3.3 Notification Service

```dart
// lib/services/notification_service.dart
import 'dart:convert';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:http/http.dart' as http;
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

class NotificationService {
  static final NotificationService _instance = NotificationService._();
  factory NotificationService() => _instance;
  NotificationService._();

  final FirebaseMessaging _fcm = FirebaseMessaging.instance;
  final FlutterLocalNotificationsPlugin _localNotifications = 
      FlutterLocalNotificationsPlugin();
  
  String? _fcmToken;

  Future<void> initialize() async {
    // Request permission
    NotificationSettings settings = await _fcm.requestPermission(
      alert: true,
      badge: true,
      sound: true,
      provisional: false,
    );

    if (settings.authorizationStatus != AuthorizationStatus.authorized) {
      print('User declined or has not granted permission');
      return;
    }

    // Initialize local notifications
    const AndroidInitializationSettings androidSettings = 
        AndroidInitializationSettings('@mipmap/ic_launcher');
    const InitializationSettings initSettings = 
        InitializationSettings(android: androidSettings);
    await _localNotifications.initialize(initSettings);

    // Get FCM token
    _fcmToken = await _fcm.getToken();
    if (_fcmToken != null) {
      await _registerToken(_fcmToken!);
      print('FCM Token: $_fcmToken');
    }

    // Listen for token refresh
    _fcm.onTokenRefresh.listen((newToken) {
      _fcmToken = newToken;
      _registerToken(newToken);
    });

    // Handle foreground messages
    FirebaseMessaging.onMessage.listen(_handleForegroundMessage);

    // Handle background messages
    FirebaseMessaging.onBackgroundMessage(_handleBackgroundMessage);

    // Handle notification tap (app opened from notification)
    FirebaseMessaging.onMessageOpenedApp.listen(_handleNotificationTap);
  }

  Future<void> _registerToken(String token) async {
    try {
      final response = await http.post(
        Uri.parse('http://YOUR_SERVER:8080/api/fcm-token'),
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer ${await _getAuthToken()}',
        },
        body: jsonEncode({
          'token': token,
          'platform': 'android', // hoặc 'ios'
        }),
      );
      
      if (response.statusCode == 201) {
        print('FCM token registered successfully');
      }
    } catch (e) {
      print('Failed to register FCM token: $e');
    }
  }

  Future<void> _handleForegroundMessage(RemoteMessage message) async {
    print('Foreground message: ${message.notification?.title}');
    
    // Hiển thị local notification khi app đang mở
    if (message.notification != null) {
      _showLocalNotification(
        title: message.notification!.title ?? '',
        body: message.notification!.body ?? '',
        data: message.data,
      );
    }
  }

  @pragma('vm:entry-point')
  static Future<void> _handleBackgroundMessage(RemoteMessage message) async {
    print('Background message: ${message.notification?.title}');
  }

  void _handleNotificationTap(RemoteMessage message) {
    print('Notification tapped: ${message.data}');
    // Navigate đến màn hình tương ứng dựa trên message.data
    _navigateBasedOnType(message.data);
  }

  void _showLocalNotification({
    required String title,
    required String body,
    Map<String, dynamic>? data,
  }) async {
    const AndroidNotificationDetails androidDetails = 
        AndroidNotificationDetails(
      'expense_tracker_channel',
      'Expense Tracker Notifications',
      channelDescription: 'Notifications from Expense Tracker',
      importance: Importance.high,
      priority: Priority.high,
    );
    
    const NotificationDetails details = 
        NotificationDetails(android: androidDetails);
    
    await _localNotifications.show(
      DateTime.now().millisecondsSinceEpoch ~/ 1000,
      title,
      body,
      details,
    );
  }

  void _navigateBasedOnType(Map<String, dynamic> data) {
    final type = data['type'];
    switch (type) {
      case 'budget_warning':
      case 'budget_exceeded':
        // Navigate to budget screen
        break;
      case 'goal_deadline':
      case 'goal_completed':
        // Navigate to goals screen
        break;
      case 'recurring_created':
        // Navigate to transactions screen
        break;
      case 'settlement':
      case 'bill_split':
        // Navigate to groups/bills screen
        break;
    }
  }

  Future<String> _getAuthToken() async {
    // Lấy auth token từ secure storage
    // Implement theo auth flow của bạn
    return '';
  }

  Future<void> deleteToken() async {
    if (_fcmToken != null) {
      try {
        await http.delete(
          Uri.parse('http://YOUR_SERVER:8080/api/fcm-token'),
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ${await _getAuthToken()}',
          },
          body: jsonEncode({'token': _fcmToken}),
        );
      } catch (e) {
        print('Failed to delete FCM token: $e');
      }
    }
  }
}
```

### 3.4 Notifications Screen

```dart
// lib/screens/notifications_screen.dart
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

class NotificationModel {
  final int id;
  final String type;
  final String title;
  final String message;
  final bool isRead;
  final DateTime createdAt;

  NotificationModel({
    required this.id,
    required this.type,
    required this.title,
    required this.message,
    required this.isRead,
    required this.createdAt,
  });

  factory NotificationModel.fromJson(Map<String, dynamic> json) {
    return NotificationModel(
      id: json['id'],
      type: json['type'],
      title: json['title'],
      message: json['message'],
      isRead: json['is_read'],
      createdAt: DateTime.parse(json['created_at']),
    );
  }

  IconData get icon {
    switch (type) {
      case 'budget_warning':
        return Icons.warning_amber;
      case 'budget_exceeded':
        return Icons.error;
      case 'goal_deadline':
        return Icons.timer;
      case 'goal_completed':
        return Icons.check_circle;
      case 'recurring_created':
        return Icons.repeat;
      case 'settlement':
        return Icons.payments;
      case 'bill_split':
        return Icons.receipt_long;
      default:
        return Icons.notifications;
    }
  }

  Color get color {
    switch (type) {
      case 'budget_warning':
        return Colors.orange;
      case 'budget_exceeded':
        return Colors.red;
      case 'goal_deadline':
        return Colors.amber;
      case 'goal_completed':
        return Colors.green;
      case 'recurring_created':
        return Colors.blue;
      case 'settlement':
        return Colors.purple;
      case 'bill_split':
        return Colors.teal;
      default:
        return Colors.grey;
    }
  }
}

class NotificationsScreen extends StatefulWidget {
  @override
  _NotificationsScreenState createState() => _NotificationsScreenState();
}

class _NotificationsScreenState extends State<NotificationsScreen> {
  List<NotificationModel> _notifications = [];
  int _unreadCount = 0;
  int _page = 1;
  bool _hasMore = true;
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    _loadNotifications();
    _loadUnreadCount();
  }

  Future<void> _loadNotifications() async {
    if (_isLoading) return;
    setState(() => _isLoading = true);

    try {
      final response = await http.get(
        Uri.parse('http://YOUR_SERVER:8080/api/notifications?page=$_page&limit=20'),
        headers: {
          'Authorization': 'Bearer ${await _getAuthToken()}',
        },
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        final List notifications = data['data'] ?? [];
        
        setState(() {
          if (_page == 1) {
            _notifications = notifications
                .map((e) => NotificationModel.fromJson(e))
                .toList();
          } else {
            _notifications.addAll(notifications
                .map((e) => NotificationModel.fromJson(e))
                .toList());
          }
          _hasMore = notifications.length == 20;
          _isLoading = false;
        });
      }
    } catch (e) {
      setState(() => _isLoading = false);
    }
  }

  Future<void> _loadUnreadCount() async {
    try {
      final response = await http.get(
        Uri.parse('http://YOUR_SERVER:8080/api/notifications/unread-count'),
        headers: {
          'Authorization': 'Bearer ${await _getAuthToken()}',
        },
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        setState(() {
          _unreadCount = data['count'];
        });
      }
    } catch (e) {}
  }

  Future<void> _markAsRead(int id) async {
    try {
      await http.put(
        Uri.parse('http://YOUR_SERVER:8080/api/notifications/$id/read'),
        headers: {
          'Authorization': 'Bearer ${await _getAuthToken()}',
        },
      );
      _loadNotifications();
      _loadUnreadCount();
    } catch (e) {}
  }

  Future<void> _markAllAsRead() async {
    try {
      await http.put(
        Uri.parse('http://YOUR_SERVER:8080/api/notifications/read-all'),
        headers: {
          'Authorization': 'Bearer ${await _getAuthToken()}',
        },
      );
      _loadNotifications();
      _loadUnreadCount();
    } catch (e) {}
  }

  Future<String> _getAuthToken() async {
    // Implement theo auth flow của bạn
    return '';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('Thông báo ($_unreadCount chưa đọc)'),
        actions: [
          if (_unreadCount > 0)
            TextButton(
              onPressed: _markAllAsRead,
              child: Text('Đọc tất cả', style: TextStyle(color: Colors.white)),
            ),
        ],
      ),
      body: _notifications.isEmpty
          ? Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.notifications_off, size: 64, color: Colors.grey),
                  SizedBox(height: 16),
                  Text('Không có thông báo nào'),
                ],
              ),
            )
          : ListView.builder(
              itemCount: _notifications.length + (_hasMore ? 1 : 0),
              itemBuilder: (context, index) {
                if (index == _notifications.length) {
                  // Load more button
                  return Center(
                    child: Padding(
                      padding: EdgeInsets.all(16),
                      child: ElevatedButton(
                        onPressed: () {
                          _page++;
                          _loadNotifications();
                        },
                        child: Text('Tải thêm'),
                      ),
                    ),
                  );
                }

                final notif = _notifications[index];
                return ListTile(
                  leading: CircleAvatar(
                    backgroundColor: notif.color.withOpacity(0.2),
                    child: Icon(notif.icon, color: notif.color, size: 20),
                  ),
                  title: Text(
                    notif.title,
                    style: TextStyle(
                      fontWeight: notif.isRead ? FontWeight.normal : FontWeight.bold,
                    ),
                  ),
                  subtitle: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(notif.message),
                      SizedBox(height: 4),
                      Text(
                        _formatDate(notif.createdAt),
                        style: TextStyle(fontSize: 12, color: Colors.grey),
                      ),
                    ],
                  ),
                  tileColor: notif.isRead ? null : Colors.blue.withOpacity(0.05),
                  onTap: () => _markAsRead(notif.id),
                );
              },
            ),
    );
  }

  String _formatDate(DateTime date) {
    final now = DateTime.now();
    final diff = now.difference(date);

    if (diff.inMinutes < 60) {
      return '${diff.inMinutes} phút trước';
    } else if (diff.inHours < 24) {
      return '${diff.inHours} giờ trước';
    } else if (diff.inDays < 7) {
      return '${diff.inDays} ngày trước';
    } else {
      return '${date.day}/${date.month}/${date.year}';
    }
  }
}
```

### 3.5 Badge Count (TabBar)

```dart
// lib/widgets/notification_badge.dart
import 'package:flutter/material.dart';
import '../services/api_service.dart';

class NotificationBadge extends StatefulWidget {
  final Widget child;

  const NotificationBadge({Key? key, required this.child}) : super(key: key);

  @override
  _NotificationBadgeState createState() => _NotificationBadgeState();
}

class _NotificationBadgeState extends State<NotificationBadge> {
  int _unreadCount = 0;

  @override
  void initState() {
    super.initState();
    _loadUnreadCount();
    // Poll mỗi 30 giây
    Future.delayed(Duration(seconds: 30), _loadUnreadCount);
  }

  Future<void> _loadUnreadCount() async {
    try {
      final response = await ApiService.get('/api/notifications/unread-count');
      if (mounted) {
        setState(() {
          _unreadCount = response['count'] ?? 0;
        });
      }
    } catch (e) {}
    
    if (mounted) {
      Future.delayed(Duration(seconds: 30), _loadUnreadCount);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_unreadCount == 0) return widget.child;
    
    return Stack(
      clipBehavior: Clip.none,
      children: [
        widget.child,
        Positioned(
          right: -4,
          top: -4,
          child: Container(
            padding: EdgeInsets.all(4),
            decoration: BoxDecoration(
              color: Colors.red,
              shape: BoxShape.circle,
            ),
            child: Text(
              _unreadCount > 99 ? '99+' : '$_unreadCount',
              style: TextStyle(
                color: Colors.white,
                fontSize: 10,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
        ),
      ],
    );
  }
}
```

### 3.6 Usage trong BottomNavigationBar

```dart
// Ví dụ sử dụng Badge trong BottomNavigationBar
BottomNavigationBar(
  items: [
    BottomNavigationBarItem(
      icon: Icon(Icons.home),
      label: 'Trang chủ',
    ),
    BottomNavigationBarItem(
      icon: NotificationBadge(
        child: Icon(Icons.notifications),
      ),
      label: 'Thông báo',
    ),
    // ... các item khác
  ],
)
```

---

## PHẦN 4: TEST

### 4.1 Test API với Postman

```bash
# 1. Login để lấy token
POST http://localhost:8080/auth/login
{
  "email": "test@example.com",
  "password": "123456"
}

# 2. Register FCM token
POST http://localhost:8080/api/fcm-token
Authorization: Bearer <token>
{
  "token": "your-fcm-token",
  "platform": "android"
}

# 3. Get notifications
GET http://localhost:8080/api/notifications
Authorization: Bearer <token>

# 4. Get unread count
GET http://localhost:8080/api/notifications/unread-count
Authorization: Bearer <token>
```

### 4.2 Test Notifications

1. Tạo expense transaction vượt 80% ngân sách → Kiểm tra notification `budget_warning`
2. Tạo expense transaction vượt 100% ngân sách → Kiểm tra notification `budget_exceeded` + email
3. Tạo goal có deadline < 7 ngày → Kiểm tra notification `goal_deadline` + email
4. Phân bổ tiền vào goal đến khi hoàn thành → Kiểm tra notification `goal_completed` + email
5. Tạo recurring transaction → Chờ scheduler chạy → Kiểm tra notification `recurring_created`
6. Tạo shared bill → Kiểm tra notification `bill_split` cho các thành viên
7. Settle debt → Kiểm tra notification `settlement` cho người nhận

---

## PHẦN 5: TROUBLESHOOTING

### FCM không hoạt động
- Kiểm tra file `firebase-service-account.json` tồn tại và đúng path
- Kiểm tra `FIREBASE_PROJECT_ID` đúng
- Kiểm tra FCM token đã đăng ký đúng

### Email không gửi được
- Kiểm tra SMTP credentials đúng
- Nếu dùng Gmail: phải bật 2FA và tạo App Password
- Kiểm tra firewall không chặn port 587

### Notification không hiển thị
- Kiểm tra user đã đăng ký FCM token
- Kiểm tra `is_active = true` trong bảng `fcm_tokens`
- Kiểm tra log server có lỗi không
