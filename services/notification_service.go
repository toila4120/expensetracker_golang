package services

import (
	"encoding/json"
	"expensetracker/models"
	"log"

	"gorm.io/gorm"
)

type NotificationService struct {
	db      *gorm.DB
	fcmSvc  *FCMService
	emailSvc *EmailService
}

func NewNotificationService(db *gorm.DB, fcmSvc *FCMService, emailSvc *EmailService) *NotificationService {
	return &NotificationService{
		db:       db,
		fcmSvc:   fcmSvc,
		emailSvc: emailSvc,
	}
}

func (s *NotificationService) CreateAndDispatch(
	userID uint,
	notifType, title, message string,
	metadata map[string]interface{},
	sendEmail bool,
	emailTo string,
	emailSubject string,
	emailBody string,
) error {
	var notif models.Notification

	if metadata != nil {
		jsonData, _ := json.Marshal(metadata)
		notif = models.Notification{
			UserID:   userID,
			Type:     notifType,
			Title:    title,
			Message:  message,
			Metadata: jsonData,
		}
	} else {
		notif = models.Notification{
			UserID:  userID,
			Type:    notifType,
			Title:   title,
			Message: message,
		}
	}

	if err := s.db.Create(&notif).Error; err != nil {
		log.Println("Create notification error:", err)
		return err
	}

	if s.fcmSvc != nil {
		data := map[string]string{
			"type": notifType,
			"id":   string(rune(notif.ID)),
		}
		if err := s.fcmSvc.SendToUser(userID, title, message, data); err != nil {
			log.Println("FCM dispatch error:", err)
		}
	}

	if sendEmail && s.emailSvc != nil && emailTo != "" {
		if err := s.emailSvc.Send(emailTo, emailSubject, emailBody); err != nil {
			log.Println("Email dispatch error:", err)
		}
	}

	return nil
}

func (s *NotificationService) GetUserNotifications(userID uint, page, limit int) ([]models.Notification, int64) {
	var notifications []models.Notification
	var total int64

	s.db.Model(&models.Notification{}).Where("user_id = ?", userID).Count(&total)

	offset := (page - 1) * limit
	s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&notifications)

	return notifications, total
}

func (s *NotificationService) GetUnreadCount(userID uint) int64 {
	var count int64
	s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count)
	return count
}

func (s *NotificationService) MarkAsRead(userID, notifID uint) error {
	return s.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Update("is_read", true).Error
}

func (s *NotificationService) MarkAllAsRead(userID uint) error {
	return s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

func (s *NotificationService) GetUserEmail(userID uint) string {
	var user models.User
	s.db.First(&user, userID)
	return user.Email
}
