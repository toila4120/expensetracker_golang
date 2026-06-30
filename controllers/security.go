package controllers

import (
	"expensetracker/models"
	"expensetracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChangePasswordInput struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword cho phép user đổi mật khẩu
func ChangePassword(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		var input ChangePasswordInput
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

		// Xác minh mật khẩu cũ
		if !utils.CheckPasswordHash(input.OldPassword, user.Password) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu cũ không đúng"})
			return
		}

		// Băm mật khẩu mới
		hashedPassword, err := utils.HashPassword(input.NewPassword)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống"})
			return
		}

		user.Password = hashedPassword
		db.Save(&user)

		ctx.JSON(http.StatusOK, gin.H{"message": "Đổi mật khẩu thành công"})
	}
}

type SetPasswordInput struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// SetPassword cho phép Google user đặt mật khẩu lần đầu
func SetPassword(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		var input SetPasswordInput
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

		// Kiểm tra user đã có password chưa
		if user.Password != "" && user.Provider == "local" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Tài khoản đã có mật khẩu. Sử dụng PUT /api/change-password để đổi mật khẩu",
			})
			return
		}

		// Băm mật khẩu mới
		hashedPassword, err := utils.HashPassword(input.NewPassword)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống"})
			return
		}

		user.Password = hashedPassword
		if err := db.Save(&user).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật mật khẩu"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"message": "Đặt mật khẩu thành công. Bạn có thể đăng nhập bằng email và mật khẩu.",
		})
	}
}
