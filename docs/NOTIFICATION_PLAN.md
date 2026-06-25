# NOTIFICATION SERVICE - KẾ HOẠCH TRIỂN KHAI

**Project:** Expense Tracker (Go + Gin)
**Frontend:** Flutter Mobile App
**Last Updated:** 2026-06-25

---

## 1. Tổng quan kiến trúc

```
Flutter App                        Go Backend (Gin)
    │                                   │
    │── POST /api/fcm-token            │  ← Flutter gửi FCM token khi login
    │                                   │     Server lưu vào DB
    │                                   │
    │── GET /api/notifications         │  ← Lấy danh sách notification
    │                                   │     Lưu trong DB, fetch khi mở app
    │                                   │
    │   (app đang mở)                  │  Event trigger → NotificationService
    │←── FCM Push ────────────────────│  ← Push qua Firebase
    │                                   │
    │   (app đang tắt)                 │  Event trigger → FCM Admin SDK
    │←── FCM Push Notification ───────│  ← Push qua Firebase
    │                                   │
    │   (user mở email)               │  Email Service → SMTP
    │←── Email cảnh báo ──────────────│  ← Email cho sự kiện quan trọng
```

## 2. Notification Types

| Type | Khi nào | Push FCM? | Email? |
|------|---------|-----------|--------|
| `budget_warning` | Chi tiêu > 80% ngân sách | Có | Không |
| `budget_exceeded` | Chi tiêu > 100% ngân sách | Có | Có |
| `goal_deadline` | Goal sắp hết hạn (<7 ngày) | Có | Có |
| `goal_completed` | Goal hoàn thành | Có | Có |
| `recurring_created` | Giao dịch định kỳ tự tạo | Có | Không |
| `settlement` | Có người trả nợ trong group | Có | Không |
| `bill_split` | Được chia hóa đơn mới | Có | Không |

## 3. Files mới cần tạo

### Models
- `models/notification.go` - Model Notification (title, type, message, is_read, metadata)
- `models/fcm_token.go` - Model lưu FCM token của user

### Services
- `services/notification_service.go` - Core logic: tạo + dispatch notification
- `services/fcm_service.go` - Gửi push qua Firebase Admin SDK
- `services/email_service.go` - Gửi email qua SMTP

### Controllers
- `controllers/notification.go` - API handlers

### Routes
- Cập nhật `routes/routes.go` - Thêm notification routes

### Config
- Cập nhật `main.go` - Khởi tạo services
- Cập nhật `.env` - Thêm Firebase + SMTP config

## 4. API mới

| Method | Endpoint | Mô tả |
|--------|----------|-------|
| `POST` | `/api/fcm-token` | Đăng ký FCM token |
| `DELETE` | `/api/fcm-token` | Xóa FCM token (khi logout) |
| `GET` | `/api/notifications` | Danh sách notification (phân trang) |
| `GET` | `/api/notifications/unread-count` | Số notification chưa đọc |
| `PUT` | `/api/notifications/:id/read` | Đánh dấu đã đọc |
| `PUT` | `/api/notifications/read-all` | Đánh dấu tất cả đã đọc |

## 5. Models chi tiết

### Notification
```go
type Notification struct {
    ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
    UserID    uint           `gorm:"not null;index" json:"user_id"`
    Type      string         `gorm:"not null" json:"type"`
    Title     string         `gorm:"not null" json:"title"`
    Message   string         `gorm:"not null" json:"message"`
    IsRead    bool           `gorm:"default:false" json:"is_read"`
    Metadata  datatypes.JSON `json:"metadata"`
    CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
}
```

### FCMToken
```go
type FCMToken struct {
    ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
    UserID    uint      `gorm:"not null;index" json:"user_id"`
    Token     string    `gorm:"uniqueIndex;not null" json:"token"`
    Platform  string    `gorm:"not null" json:"platform"` // android, ios
    IsActive  bool      `gorm:"default:true" json:"is_active"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
```

## 6. Dependency mới

```go
// go.mod - Thêm Firebase Admin SDK
require (
    firebase.google.com/go/v4 v4.14.0
)
```

## 7. Environment variables mới

```env
# Firebase
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_CREDENTIALS=./firebase-service-account.json

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
SMTP_FROM=ExpenseTracker <no-reply@expensetracker.com>
```

## 8. Chi tiết triển khai

### 8.1 FCM Service
```go
type FCMService struct {
    app *firebase.App
}

func NewFCMService() (*FCMService, error) {
    // Khởi tạo Firebase Admin SDK từ service account key
    // opt := option.WithCredentialsFile("path/to/serviceAccountKey.json")
    // app, err := firebase.NewApp(context.Background(), nil, opt)
}

func (s *FCMService) SendToUser(userID uint, title, body string, data map[string]string) error {
    // 1. Lấy FCM tokens của user từ DB
    // 2. Gửi push notification đến tất cả device
}
```

### 8.2 Notification Service
```go
type NotificationService struct {
    db       *gorm.DB
    fcmSvc   *FCMService
    emailSvc *EmailService
}

func (s *NotificationService) CreateAndDispatch(
    userID uint,
    notifType, title, message string,
    metadata map[string]interface{},
    sendEmail bool,
) error {
    // 1. Lưu notification vào DB
    // 2. Gửi FCM push notification
    // 3. Gửi email nếu sendEmail = true
}

func (s *NotificationService) GetUserNotifications(userID uint, page, limit int) ([]Notification, int64)
func (s *NotificationService) MarkAsRead(userID, notifID uint) error
func (s *NotificationService) MarkAllAsRead(userID uint) error
func (s *NotificationService) GetUnreadCount(userID uint) int64
```

### 8.3 Email Service
```go
type EmailService struct {
    host, username, password, from string
    port int
}

func NewEmailService() *EmailService {
    // Đọc từ env: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM
}

func (s *EmailService) Send(to, subject, body string) error {
    // Gửi email qua SMTP
}
```

### 8.4 Tích hợp vào Controllers hiện tại

**controllers/budget.go** - Thêm sau khi tạo expense transaction:
```go
// Kiểm tra chi tiêu vượt ngân sách
if spent > budget.Amount * 80 / 100 {
    notificationService.CreateAndDispatch(userID, "budget_warning",
        "Cảnh báo ngân sách",
        fmt.Sprintf("Chi tiêu danh mục %s đã vượt 80%% ngân sách", category),
        nil, false)
}
if spent > budget.Amount {
    notificationService.CreateAndDispatch(userID, "budget_exceeded",
        "Vượt ngân sách",
        fmt.Sprintf("Chi tiêu danh mục %s đã vượt ngân sách", category),
        nil, true) // Gửi email
}
```

**controllers/financial_goal.go** - Thêm khi kiểm tra goal:
```go
// Goal sắp hết hạn
if daysLeft < 7 {
    notificationService.CreateAndDispatch(userID, "goal_deadline",
        "Mục tiêu sắp hết hạn",
        fmt.Sprintf("Mục tiêu %s còn %d ngày", goal.Name, daysLeft),
        nil, true)
}

// Goal hoàn thành
if goal.CurrentAmount >= goal.TargetAmount {
    notificationService.CreateAndDispatch(userID, "goal_completed",
        "Mục tiêu hoàn thành!",
        fmt.Sprintf("Mục tiêu %s đã hoàn thành", goal.Name),
        nil, true)
}
```

**scheduler/scheduler.go** - Thêm khi tạo recurring transaction:
```go
notificationService.CreateAndDispatch(recurring.UserID, "recurring_created",
    "Giao dịch định kỳ",
    fmt.Sprintf("Đã tạo giao dịch tự động: %s %d VND", recurring.Type, recurring.Amount),
    nil, false)
```

## 9. Flutter Integration

### Đăng ký FCM Token
```dart
// lib/services/notification_service.dart
import 'package:firebase_messaging/firebase_messaging.dart';

class NotificationService {
  final FirebaseMessaging _fcm = FirebaseMessaging.instance;

  Future<void> initialize() async {
    // Yêu cầu quyền
    NotificationSettings settings = await _fcm.requestPermission(
      alert: true,
      badge: true,
      sound: true,
    );

    if (settings.authorizationStatus == AuthorizationStatus.authorized) {
      // Lấy FCM token
      String? token = await _fcm.getToken();
      if (token != null) {
        await _registerToken(token);
      }

      // Lắng nghe token refresh
      _fcm.onTokenRefresh.listen((newToken) {
        _registerToken(newToken);
      });
    }

    // Xử lý notification khi app đang mở
    FirebaseMessaging.onMessage.listen(_handleForegroundMessage);

    // Xử lý khi app được mở từ notification
    FirebaseMessaging.onMessageOpenedApp.listen(_handleNotificationTap);
  }

  Future<void> _registerToken(String token) async {
    // POST /api/fcm-token với token
    await api.post('/api/fcm-token', body: {'token': token});
  }

  void _handleForegroundMessage(RemoteMessage message) {
    // Hiển thị notification trong app (Snackbar, Toast, etc.)
    print('Foreground message: ${message.notification?.title}');
  }

  void _handleNotificationTap(RemoteMessage message) {
    // Navigate đến màn hình tương ứng
    print('Notification tapped: ${message.data}');
  }
}
```

### Hiển thị In-app Notifications
```dart
// lib/screens/notifications_screen.dart
class NotificationsScreen extends StatefulWidget {
  @override
  _NotificationsScreenState createState() => _NotificationsScreenState();
}

class _NotificationsScreenState extends State<NotificationsScreen> {
  List<Notification> _notifications = [];
  int _unreadCount = 0;

  @override
  void initState() {
    super.initState();
    _loadNotifications();
    _loadUnreadCount();
  }

  Future<void> _loadNotifications() async {
    final response = await api.get('/api/notifications?page=1&limit=20');
    setState(() {
      _notifications = (response['data'] as List)
          .map((e) => Notification.fromJson(e))
          .toList();
    });
  }

  Future<void> _loadUnreadCount() async {
    final response = await api.get('/api/notifications/unread-count');
    setState(() {
      _unreadCount = response['count'];
    });
  }

  Future<void> _markAsRead(int id) async {
    await api.put('/api/notifications/$id/read');
    _loadNotifications();
    _loadUnreadCount();
  }

  Future<void> _markAllAsRead() async {
    await api.put('/api/notifications/read-all');
    _loadNotifications();
    _loadUnreadCount();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('Thông báo ($_unreadCount)'),
        actions: [
          TextButton(
            onPressed: _markAllAsRead,
            child: Text('Đọc tất cả'),
          ),
        ],
      ),
      body: ListView.builder(
        itemCount: _notifications.length,
        itemBuilder: (context, index) {
          final notif = _notifications[index];
          return ListTile(
            leading: Icon(
              notif.isRead ? Icons.notifications_off : Icons.notifications,
              color: notif.isRead ? Colors.grey : Colors.blue,
            ),
            title: Text(notif.title),
            subtitle: Text(notif.message),
            trailing: Text(notif.createdAt.toString()),
            onTap: () => _markAsRead(notif.id),
          );
        },
      ),
    );
  }
}
```

## 10. Sắp xếp thứ tự triển khai

| Bước | File | Mô tả |
|------|------|-------|
| 1 | `models/notification.go` | Tạo model + AutoMigrate |
| 2 | `models/fcm_token.go` | Tạo model FCM token |
| 3 | `services/fcm_service.go` | Firebase Admin SDK |
| 4 | `services/email_service.go` | SMTP sender |
| 5 | `services/notification_service.go` | Core logic |
| 6 | `controllers/notification.go` | API handlers |
| 7 | `routes/routes.go` | Thêm routes |
| 8 | `main.go` | Khởi tạo services |
| 9 | Tích hợp event triggers | Budget, Goal, Scheduler |
| 10 | Flutter integration | FCM + Notification screen |

## 11. Firebase Setup

1. Tạo Firebase project tại https://console.firebase.google.com
2. Thêm Android app (com.example.expensetracker)
3. Download `google-services.json` → đặt vào `android/app/`
4. Tạo Service Account Key:
   - Firebase Console → Project Settings → Service Accounts
   - Generate new private key
   - Lưu file JSON vào project (thêm vào .gitignore)
5. Cấu hình Firebase Admin SDK trong Go backend

## 12. Ghi chú

- FCM token cần được refresh định kỳ (Flutter tự động xử lý)
- Nên lưu nhiều FCM token per user (mỗi device 1 token)
- Khi user logout → xóa FCM token
- Email chỉ gửi cho sự kiện quan trọng (budget_exceeded, goal_deadline, goal_completed)
- Notification metadata có thể chứa JSON để Flutter navigate đúng màn hình
