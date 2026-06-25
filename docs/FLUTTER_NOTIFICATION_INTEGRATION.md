# TÍCH HỢP NOTIFICATION VÀO FLUTTER APP

**Cập nhật:** 2026-06-25

---

## 1. CÀI ĐẶT DEPENDENCIES

```yaml
# pubspec.yaml
dependencies:
  firebase_core: ^3.8.1
  firebase_messaging: ^15.2.1
  flutter_local_notifications: ^18.0.1
  http: ^1.2.0
  shared_preferences: ^2.2.0
```

```bash
flutter pub get
```

---

## 2. FIREBASE CONFIG

### Android

```yaml
# android/app/build.gradle
dependencies {
    implementation platform('com.google.firebase:firebase-bom:33.7.0')
    implementation 'com.google.firebase:firebase-messaging'
}
```

Copy `google-services.json` vào `android/app/`

### iOS

```bash
cd ios && pod install
```

---

## 3. MAIN.DART

```dart
// lib/main.dart
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'services/notification_service.dart';

// Background message handler (phải ở top-level)
@pragma('vm:entry-point')
Future<void> _firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  await Firebase.initializeApp();
  print('Background message: ${message.messageId}');
}

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await Firebase.initializeApp();
  
  // Register background handler
  FirebaseMessaging.onBackgroundMessage(_firebaseMessagingBackgroundHandler);
  
  // Initialize notification service
  await NotificationService().initialize();
  
  runApp(MyApp());
}
```

---

## 4. NOTIFICATION SERVICE

```dart
// lib/services/notification_service.dart
import 'dart:convert';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

class NotificationService {
  static final NotificationService _instance = NotificationService._();
  factory NotificationService() => _instance;
  NotificationService._();

  final FirebaseMessaging _fcm = FirebaseMessaging.instance;
  final FlutterLocalNotificationsPlugin _localNotifications = 
      FlutterLocalNotificationsPlugin();
  
  String? _fcmToken;
  String? _authToken;
  Function(Map<String, dynamic>)? onNotificationTap;

  // ========================
  // KHỞI TẠO
  // ========================
  Future<void> initialize() async {
    // Lấy auth token từ storage
    await _loadAuthToken();
    
    // Request permission
    NotificationSettings settings = await _fcm.requestPermission(
      alert: true,
      badge: true,
      sound: true,
      provisional: false,
    );

    if (settings.authorizationStatus != AuthorizationStatus.authorized) {
      print('User declined permission');
      return;
    }

    // Initialize local notifications
    const AndroidInitializationSettings androidSettings = 
        AndroidInitializationSettings('@mipmap/ic_launcher');
    const InitializationSettings initSettings = 
        InitializationSettings(android: androidSettings);
    await _localNotifications.initialize(
      initSettings,
      onDidReceiveNotificationResponse: (details) {
        // Xử lý khi tap vào notification
        if (details.payload != null) {
          final data = jsonDecode(details.payload!);
          onNotificationTap?.call(data);
        }
      },
    );

    // Create notification channel (Android)
    await _createNotificationChannel();

    // Get FCM token
    _fcmToken = await _fcm.getToken();
    if (_fcmToken != null) {
      print('FCM Token: $_fcmToken');
      await _registerToken(_fcmToken!);
    }

    // Listen for token refresh
    _fcm.onTokenRefresh.listen((newToken) {
      _fcmToken = newToken;
      _registerToken(newToken);
    });

    // Handle foreground messages
    FirebaseMessaging.onMessage.listen(_handleForegroundMessage);

    // Handle notification tap (app opened from background)
    FirebaseMessaging.onMessageOpenedApp.listen(_handleNotificationTap);

    // Check if app opened from notification (terminated state)
    final initialMessage = await _fcm.getInitialMessage();
    if (initialMessage != null) {
      _handleNotificationTap(initialMessage);
    }
  }

  // ========================
  // TẠO NOTIFICATION CHANNEL
  // ========================
  Future<void> _createNotificationChannel() async {
    const AndroidNotificationChannel channel = AndroidNotificationChannel(
      'expense_tracker_channel',
      'Expense Tracker Notifications',
      description: 'Thông báo từ ứng dụng Expense Tracker',
      importance: Importance.high,
      enableVibration: true,
    );

    await _localNotifications
        .resolvePlatformSpecificImplementation<
            AndroidFlutterLocalNotificationsPlugin>()
        ?.createNotificationChannel(channel);
  }

  // ========================
  // ĐĂNG KÝ FCM TOKEN
  // ========================
  Future<void> _registerToken(String token) async {
    if (_authToken == null) return;

    try {
      final response = await http.post(
        Uri.parse('https://your-app.onrender.com/api/fcm-token'),
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $_authToken',
        },
        body: jsonEncode({
          'token': token,
          'platform': _getPlatform(),
        }),
      );
      
      if (response.statusCode == 201) {
        print('FCM token registered');
      }
    } catch (e) {
      print('Failed to register token: $e');
    }
  }

  // ========================
  // XÓA FCM TOKEN (LOGOUT)
  // ========================
  Future<void> deleteToken() async {
    if (_fcmToken == null || _authToken == null) return;

    try {
      await http.delete(
        Uri.parse('https://your-app.onrender.com/api/fcm-token'),
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $_authToken',
        },
        body: jsonEncode({'token': _fcmToken}),
      );
      print('FCM token deleted');
    } catch (e) {
      print('Failed to delete token: $e');
    }
  }

  // ========================
  // XỬ LÝ MESSAGE
  // ========================
  void _handleForegroundMessage(RemoteMessage message) {
    print('Foreground: ${message.notification?.title}');
    
    if (message.notification != null) {
      _showLocalNotification(
        id: message.hashCode,
        title: message.notification!.title ?? '',
        body: message.notification!.body ?? '',
        data: message.data,
      );
    }
  }

  void _handleNotificationTap(RemoteMessage message) {
    print('Tapped: ${message.data}');
    onNotificationTap?.call(message.data);
  }

  // ========================
  // HIỂN THỊ LOCAL NOTIFICATION
  // ========================
  Future<void> _showLocalNotification({
    required int id,
    required String title,
    required String body,
    Map<String, dynamic>? data,
  }) async {
    const AndroidNotificationDetails androidDetails = 
        AndroidNotificationDetails(
      'expense_tracker_channel',
      'Expense Tracker Notifications',
      channelDescription: 'Thông báo từ Expense Tracker',
      importance: Importance.high,
      priority: Priority.high,
      enableVibration: true,
      icon: '@mipmap/ic_launcher',
    );
    
    const NotificationDetails details = 
        NotificationDetails(android: androidDetails);
    
    await _localNotifications.show(
      id,
      title,
      body,
      details,
      payload: data != null ? jsonEncode(data) : null,
    );
  }

  // ========================
  // UTILS
  // ========================
  String _getPlatform() {
    return 'android'; // hoặc 'ios' tùy platform
  }

  Future<void> _loadAuthToken() async {
    final prefs = await SharedPreferences.getInstance();
    _authToken = prefs.getString('auth_token');
  }

  Future<void> setAuthToken(String token) async {
    _authToken = token;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('auth_token', token);
    
    // Re-register token nếu có
    if (_fcmToken != null) {
      await _registerToken(_fcmToken!);
    }
  }
}
```

---

## 5. NOTIFICATION SCREEN

```dart
// lib/screens/notifications_screen.dart
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import '../services/notification_service.dart';

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
      case 'budget_warning': return Icons.warning_amber;
      case 'budget_exceeded': return Icons.error;
      case 'goal_deadline': return Icons.timer;
      case 'goal_completed': return Icons.check_circle;
      case 'recurring_created': return Icons.repeat;
      case 'settlement': return Icons.payments;
      case 'bill_split': return Icons.receipt_long;
      default: return Icons.notifications;
    }
  }

  Color get color {
    switch (type) {
      case 'budget_warning': return Colors.orange;
      case 'budget_exceeded': return Colors.red;
      case 'goal_deadline': return Colors.amber;
      case 'goal_completed': return Colors.green;
      case 'recurring_created': return Colors.blue;
      case 'settlement': return Colors.purple;
      case 'bill_split': return Colors.teal;
      default: return Colors.grey;
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
  String? _authToken;

  @override
  void initState() {
    super.initState();
    _loadAuthToken();
  }

  Future<void> _loadAuthToken() async {
    final prefs = await SharedPreferences.getInstance();
    _authToken = prefs.getString('auth_token');
    if (_authToken != null) {
      _loadNotifications();
      _loadUnreadCount();
    }
  }

  Future<void> _loadNotifications() async {
    if (_isLoading || _authToken == null) return;
    setState(() => _isLoading = true);

    try {
      final response = await http.get(
        Uri.parse('https://your-app.onrender.com/api/notifications?page=$_page&limit=20'),
        headers: {'Authorization': 'Bearer $_authToken'},
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
            _notifications.addAll(
              notifications.map((e) => NotificationModel.fromJson(e)).toList(),
            );
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
    if (_authToken == null) return;
    
    try {
      final response = await http.get(
        Uri.parse('https://your-app.onrender.com/api/notifications/unread-count'),
        headers: {'Authorization': 'Bearer $_authToken'},
      );

      if (response.statusCode == 200) {
        setState(() {
          _unreadCount = jsonDecode(response.body)['count'];
        });
      }
    } catch (e) {}
  }

  Future<void> _markAsRead(int id) async {
    if (_authToken == null) return;
    
    try {
      await http.put(
        Uri.parse('https://your-app.onrender.com/api/notifications/$id/read'),
        headers: {'Authorization': 'Bearer $_authToken'},
      );
      _loadNotifications();
      _loadUnreadCount();
    } catch (e) {}
  }

  Future<void> _markAllAsRead() async {
    if (_authToken == null) return;
    
    try {
      await http.put(
        Uri.parse('https://your-app.onrender.com/api/notifications/read-all'),
        headers: {'Authorization': 'Bearer $_authToken'},
      );
      _loadNotifications();
      _loadUnreadCount();
    } catch (e) {}
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
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_notifications.isEmpty && !_isLoading) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.notifications_off, size: 64, color: Colors.grey),
            SizedBox(height: 16),
            Text('Không có thông báo nào'),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () async {
        _page = 1;
        await _loadNotifications();
        await _loadUnreadCount();
      },
      child: ListView.builder(
        itemCount: _notifications.length + (_hasMore ? 1 : 0),
        itemBuilder: (context, index) {
          if (index == _notifications.length) {
            return Center(
              child: Padding(
                padding: EdgeInsets.all(16),
                child: _isLoading
                    ? CircularProgressIndicator()
                    : ElevatedButton(
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
                Text(notif.message, maxLines: 2, overflow: TextOverflow.ellipsis),
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

    if (diff.inMinutes < 60) return '${diff.inMinutes} phút trước';
    if (diff.inHours < 24) return '${diff.inHours} giờ trước';
    if (diff.inDays < 7) return '${diff.inDays} ngày trước';
    return '${date.day}/${date.month}/${date.year}';
  }
}
```

---

## 6. NOTIFICATION BADGE WIDGET

```dart
// lib/widgets/notification_badge.dart
import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http';
import 'package:shared_preferences/shared_preferences.dart';

class NotificationBadge extends StatefulWidget {
  final Widget child;
  final VoidCallback? onTap;

  const NotificationBadge({
    Key? key, 
    required this.child,
    this.onTap,
  }) : super(key: key);

  @override
  _NotificationBadgeState createState() => _NotificationBadgeState();
}

class _NotificationBadgeState extends State<NotificationBadge> {
  int _unreadCount = 0;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _loadUnreadCount();
    // Poll mỗi 30 giây
    _timer = Timer.periodic(Duration(seconds: 30), (_) => _loadUnreadCount());
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _loadUnreadCount() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final token = prefs.getString('auth_token');
      if (token == null) return;

      final response = await http.get(
        Uri.parse('https://your-app.onrender.com/api/notifications/unread-count'),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode == 200 && mounted) {
        setState(() {
          _unreadCount = jsonDecode(response.body)['count'] ?? 0;
        });
      }
    } catch (e) {}
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: widget.onTap,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          widget.child,
          if (_unreadCount > 0)
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
      ),
    );
  }
}
```

---

## 7. NAVIGATE THEO NOTIFICATION TYPE

```dart
// lib/utils/notification_navigator.dart
class NotificationNavigator {
  static void navigate(BuildContext context, Map<String, dynamic> data) {
    final type = data['type'];
    
    switch (type) {
      case 'budget_warning':
      case 'budget_exceeded':
        Navigator.pushNamed(context, '/budgets');
        break;
        
      case 'goal_deadline':
      case 'goal_completed':
        Navigator.pushNamed(context, '/goals');
        break;
        
      case 'recurring_created':
        Navigator.pushNamed(context, '/transactions');
        break;
        
      case 'settlement':
      case 'bill_split':
        Navigator.pushNamed(context, '/groups');
        break;
        
      default:
        Navigator.pushNamed(context, '/notifications');
    }
  }
}
```

---

## 8. SỬ DỤNG TRONG APP

### Main.dart

```dart
class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Expense Tracker',
      initialRoute: '/',
      routes: {
        '/': (context) => HomeScreen(),
        '/notifications': (context) => NotificationsScreen(),
        '/budgets': (context) => BudgetsScreen(),
        '/goals': (context) => GoalsScreen(),
        '/transactions': (context) => TransactionsScreen(),
        '/groups': (context) => GroupsScreen(),
      },
    );
  }
}
```

### HomeScreen với Badge

```dart
class HomeScreen extends StatefulWidget {
  @override
  _HomeScreenState createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _currentIndex = 0;

  final List<Widget> _screens = [
    DashboardScreen(),
    TransactionsScreen(),
    NotificationsScreen(),
    ProfileScreen(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: _screens[_currentIndex],
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _currentIndex,
        onTap: (index) => setState(() => _currentIndex = index),
        items: [
          BottomNavigationBarItem(icon: Icon(Icons.home), label: 'Trang chủ'),
          BottomNavigationBarItem(icon: Icon(Icons.receipt), label: 'Giao dịch'),
          BottomNavigationBarItem(
            icon: NotificationBadge(
              child: Icon(Icons.notifications),
            ),
            label: 'Thông báo',
          ),
          BottomNavigationBarItem(icon: Icon(Icons.person), label: 'Cá nhân'),
        ],
      ),
    );
  }
}
```

### Login sau khi login thành công

```dart
// Sau khi login thành công
Future<void> _login(String email, String password) async {
  final response = await http.post(
    Uri.parse('https://your-app.onrender.com/auth/login'),
    body: {'email': email, 'password': password},
  );

  if (response.statusCode == 200) {
    final data = jsonDecode(response.body);
    final token = data['access_token'];
    
    // Lưu token
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('auth_token', token);
    
    // Đăng ký FCM token
    await NotificationService().setAuthToken(token);
    
    Navigator.pushReplacementNamed(context, '/');
  }
}
```

### Logout

```dart
Future<void> _logout() async {
  // Xóa FCM token trước khi logout
  await NotificationService().deleteToken();
  
  final prefs = await SharedPreferences.getInstance();
  await prefs.clear();
  
  Navigator.pushReplacementNamed(context, '/login');
}
```

---

## 9. TEST TRÊN DEVICE THẬT

```dart
// lib/main.dart - thêm test mode
void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await Firebase.initializeApp();
  
  // Test: In FCM token để đăng ký
  final token = await FirebaseMessaging.instance.getToken();
  print('=============================');
  print('FCM TOKEN: $token');
  print('=============================');
  
  await NotificationService().initialize();
  runApp(MyApp());
}
```

Lấy token từ console → Đăng ký qua API → Tạo notification → Kiểm tra push notification hiển thị trên device.

---

## 10. CHECKLIST

| Mục | Trạng thái |
|-----|------------|
| Thêm firebase_core, firebase_messaging | ☐ |
| Copy google-services.json vào android/app | ☐ |
| Initialize Firebase trong main.dart | ☐ |
| Tạo NotificationService | ☐ |
| Request permission khi login | ☐ |
| Register FCM token sau login | ☐ |
| Delete FCM token khi logout | ☐ |
| Hiển thị badge unread count | ☐ |
| NotificationsScreen | ☐ |
| Navigate theo notification type | ☐ |
