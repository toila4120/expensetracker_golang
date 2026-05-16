package controllers

import (
	"expensetracker/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RecurringInput struct {
	Amount     int    `json:"amount" binding:"required,gt=0"`
	Category   string `json:"category" binding:"required"`
	Type       string `json:"type" binding:"required,oneof=income expense"`
	Note       string `json:"note"`
	DayOfMonth int    `json:"day_of_month" binding:"required,min=1,max=31"`
}

// CreateRecurring tạo giao dịch định kỳ
func CreateRecurring(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		var input RecurringInput
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		recurring := models.RecurringTransaction{
			UserID:     userID,
			Amount:     input.Amount,
			Category:   input.Category,
			Type:       input.Type,
			Note:       input.Note,
			DayOfMonth: input.DayOfMonth,
			IsActive:   true,
		}

		if err := db.Create(&recurring).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo giao dịch định kỳ"})
			return
		}

		ctx.JSON(http.StatusCreated, gin.H{
			"message": "Tạo giao dịch định kỳ thành công",
			"data":    recurring,
		})
	}
}

// GetRecurrings lấy danh sách giao dịch định kỳ
func GetRecurrings(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		var recurrings []models.RecurringTransaction
		db.Where("user_id = ?", userID).Find(&recurrings)

		ctx.JSON(http.StatusOK, gin.H{
			"count": len(recurrings),
			"data":  recurrings,
		})
	}
}

// ToggleRecurring bật/tắt giao dịch định kỳ
func ToggleRecurring(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)
		recurringID, _ := strconv.Atoi(ctx.Param("id"))

		var recurring models.RecurringTransaction
		if err := db.Where("id = ? AND user_id = ?", recurringID, userID).First(&recurring).Error; err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy giao dịch định kỳ"})
			return
		}

		recurring.IsActive = !recurring.IsActive
		db.Save(&recurring)

		status := "bật"
		if !recurring.IsActive {
			status = "tắt"
		}

		ctx.JSON(http.StatusOK, gin.H{
			"message": "Đã " + status + " giao dịch định kỳ",
			"data":    recurring,
		})
	}
}

// DeleteRecurring xóa giao dịch định kỳ
func DeleteRecurring(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)
		recurringID := ctx.Param("id")

		var recurring models.RecurringTransaction
		if err := db.Where("id = ? AND user_id = ?", recurringID, userID).First(&recurring).Error; err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy giao dịch định kỳ"})
			return
		}

		db.Delete(&recurring)
		ctx.JSON(http.StatusOK, gin.H{"message": "Xóa giao dịch định kỳ thành công"})
	}
}
