// File: controllers/auth.go
// Mục tiêu: Xử lý logic đăng ký và đăng nhập của người dùng.
//
// Nhiệm vụ của bạn:
// 1. Viết hàm Register(c *gin.Context):
//    - Nhận dữ liệu (email, password) từ request body (JSON).
//    - Kiểm tra email đã tồn tại trong DB chưa.
//    - Băm mật khẩu (dùng utils.HashPassword).
//    - Khởi tạo model User và lưu user vào DB.
//    - Trả về JSON thông báo thành công.
// 2. Viết hàm Login(c *gin.Context):
//    - Nhận email/password từ request body.
//    - Tìm user trong DB theo email. Nếu không thấy báo lỗi.
//    - Kiểm tra mật khẩu (dùng utils.CheckPasswordHash). Nếu sai báo lỗi.
//    - Tạo JWT (dùng utils.GenerateToken truyền userID vào).
//    - Trả về chuỗi JWT token cho client.
//
// Kiến thức cần học:
// - Dùng c.ShouldBindJSON trong Gin để parse dữ liệu request sang Struct.
// - Cách trả về response JSON với c.JSON.
// - Tìm hiểu các HTTP Status Codes (200 OK, 400 Bad Request cho dữ liệu sai, 401 Unauthorized khi login sai, 201 Created).

package controllers

import (
	"expensetracker/models"
	"expensetracker/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input AuthInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu đầu vào không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		var existingUser models.User
		if err := db.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email đã tồn tại",
			})
			return
		}

		hashedPassword, err := utils.HashPassword(input.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống khi xử lý mật khẩu",
			})
			return
		}

		username := strings.Split(input.Email, "@")[0]

		newUser := models.User{
			Username: username,
			Email:    input.Email,
			Password: hashedPassword,
		}

		if err := db.Create(&newUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể tạo tài khoản",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Đăng ký thành công",
		})
	}

}

func Login(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input AuthInput
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu đầu vào không hợp lệ",
				"details": err.Error(),
			})
			return
		}
		var existingUser models.User
		if err := db.Where("email = ?", input.Email).First(&existingUser).Error; err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": "Email hoặc mật khẩu không đúng",
			})
			return
		}

		isMatch := utils.CheckPasswordHash(input.Password, existingUser.Password)
		if !isMatch {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": "Email hoặc mật khẩu không đúng",
			})
			return
		}

		accessToken, err := utils.GenerateAccessToken(existingUser.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống khi tạo Access Token",
			})
			return
		}

		refreshToken, err := utils.GenerateRefreshToken(existingUser.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống khi tạo Refresh Token",
			})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"message":       "Đăng nhập thành công",
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

// Hàm RefreshToken
func RefreshToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input RefreshTokenInput

		// 1. Lấy refresh token từ request body
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Vui lòng cung cấp refresh token",
				"details": err.Error(),
			})
			return
		}

		// 2. Xác thực Refresh Token và lấy ra UserID
		// Giả sử bạn có hàm utils.ValidateToken để giải mã và kiểm tra token
		userID, err := utils.ValidateToken(input.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Refresh token không hợp lệ hoặc đã hết hạn",
			})
			return
		}

		// 3. (Tùy chọn nhưng khuyến nghị) Kiểm tra xem user có còn tồn tại trong DB không
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Người dùng không tồn tại hoặc đã bị xóa",
			})
			return
		}

		// 4. Tạo Access Token mới
		newToken, err := utils.GenerateAccessToken(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống khi tạo Access Token mới",
			})
			return
		}

		// 5. Trả về token mới cho client
		c.JSON(http.StatusOK, gin.H{
			"message":      "Làm mới token thành công",
			"access_token": newToken,
		})
	}
}
