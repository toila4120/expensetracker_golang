package controllers

import (
	"expensetracker/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetProfile trả về thông tin cá nhân của user
func GetProfile(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"provider":    user.Provider,
			"has_password": user.Password != "",
			"created_at":  user.CreatedAt,
		})
	}
}

type UpdateProfileInput struct {
	Username string `json:"username" binding:"required"`
}

// UpdateProfile cập nhật thông tin cá nhân
func UpdateProfile(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		var input UpdateProfileInput
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
			return
		}

		// Chỉ cập nhật trường username thay vì dùng db.Save để tránh lỗi timezone/định dạng trên PostgreSQL
		if err := db.Model(&user).Update("username", input.Username).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Không thể cập nhật thông tin",
				"details": err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"message": "Cập nhật thành công",
			"data": gin.H{
				"id":         user.ID,
				"username":   user.Username,
				"email":      user.Email,
				"created_at": user.CreatedAt,
			},
		})
	}
}
