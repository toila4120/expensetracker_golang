package controllers

import (
	"expensetracker/models"
	"expensetracker/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RegisterFCMTokenInput struct {
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=android ios"`
}

func RegisterFCMToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input RegisterFCMTokenInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		userID := c.MustGet("currentUserID").(uint)

		var existing models.FCMToken
		if err := db.Where("token = ?", input.Token).First(&existing).Error; err == nil {
			existing.IsActive = true
			existing.UserID = userID
			existing.Platform = input.Platform
			db.Save(&existing)
			c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật FCM token"})
			return
		}

		fcmToken := models.FCMToken{
			UserID:   userID,
			Token:    input.Token,
			Platform: input.Platform,
			IsActive: true,
		}

		if err := db.Create(&fcmToken).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu FCM token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Đã đăng ký FCM token thành công"})
	}
}

func DeleteFCMToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Token string `json:"token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Token không hợp lệ"})
			return
		}

		userID := c.MustGet("currentUserID").(uint)

		if err := db.Where("user_id = ? AND token = ?", userID, input.Token).
			Delete(&models.FCMToken{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa FCM token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Đã xóa FCM token"})
	}
}

func GetNotifications(db *gorm.DB, notifSvc *services.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}

		notifications, total := notifSvc.GetUserNotifications(userID, page, limit)

		c.JSON(http.StatusOK, gin.H{
			"data":  notifications,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}

func GetUnreadCount(db *gorm.DB, notifSvc *services.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		count := notifSvc.GetUnreadCount(userID)
		c.JSON(http.StatusOK, gin.H{"count": count})
	}
}

func MarkNotificationAsRead(db *gorm.DB, notifSvc *services.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		notifIDStr := c.Param("id")
		notifID, err := strconv.ParseUint(notifIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
			return
		}

		if err := notifSvc.MarkAsRead(userID, uint(notifID)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu đã đọc"})
	}
}

func MarkAllNotificationsAsRead(db *gorm.DB, notifSvc *services.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)

		if err := notifSvc.MarkAllAsRead(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Đã đánh dấu tất cả đã đọc"})
	}
}
