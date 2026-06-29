package controllers

import (
	"expensetracker/models"
	"expensetracker/services"
	"expensetracker/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GoogleLoginInput struct {
	IDToken      string `json:"id_token" binding:"required"`
	AccessToken  string `json:"access_token"`
}

func GoogleLogin(db *gorm.DB, googleOAuth *services.GoogleOAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input GoogleLoginInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu đầu vào không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		// Verify Google token
		var googleUser *services.GoogleUserInfo
		var err error

		if input.IDToken != "" {
			// Verify ID token
			googleUser, err = googleOAuth.VerifyToken(input.IDToken)
		} else if input.AccessToken != "" {
			// Verify access token
			googleUser, err = googleOAuth.VerifyAccessToken(input.AccessToken)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Cần cung cấp id_token hoặc access_token",
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token Google không hợp lệ",
			})
			return
		}

		// Check email verified
		if !googleUser.VerifiedEmail {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Email Google chưa được xác thực",
			})
			return
		}

		// Extract username from email
		username := strings.Split(googleUser.Email, "@")[0]

		// Find existing user by email
		var existingUser models.User
		err = db.Where("email = ?", googleUser.Email).First(&existingUser).Error

		if err == gorm.ErrRecordNotFound {
			// User doesn't exist - create new account
			newUser := models.User{
				Username:   username,
				Email:      googleUser.Email,
				Password:   "", // No password for Google login
				Provider:   "google",
				ProviderID: googleUser.ID,
			}

			if err := db.Create(&newUser).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Không thể tạo tài khoản",
				})
				return
			}

			// Generate tokens
			accessToken, err := utils.GenerateAccessToken(newUser.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Lỗi hệ thống khi tạo Access Token",
				})
				return
			}

			refreshToken, err := utils.GenerateRefreshToken(newUser.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Lỗi hệ thống khi tạo Refresh Token",
				})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"message":       "Đăng ký thành công bằng Google",
				"access_token":  accessToken,
				"refresh_token": refreshToken,
				"is_new_user":   true,
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống",
			})
			return
		}

		// User exists - check if already linked to Google
		if existingUser.Provider == "google" && existingUser.ProviderID == googleUser.ID {
			// Already linked - just login
			accessToken, err := utils.GenerateAccessToken(existingUser.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Lỗi hệ thống khi tạo Access Token",
				})
				return
			}

			refreshToken, err := utils.GenerateRefreshToken(existingUser.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Lỗi hệ thống khi tạo Refresh Token",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message":       "Đăng nhập thành công",
				"access_token":  accessToken,
				"refresh_token": refreshToken,
				"is_new_user":   false,
			})
			return
		}

		// User exists with same email but different provider (local account)
		// Link Google to existing account
		existingUser.Provider = "google"
		existingUser.ProviderID = googleUser.ID

		if err := db.Save(&existingUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể liên kết tài khoản Google",
			})
			return
		}

		// Generate tokens
		accessToken, err := utils.GenerateAccessToken(existingUser.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống khi tạo Access Token",
			})
			return
		}

		refreshToken, err := utils.GenerateRefreshToken(existingUser.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống khi tạo Refresh Token",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "Đăng nhập thành công và đã liên kết tài khoản Google",
			"access_token":   accessToken,
			"refresh_token":  refreshToken,
			"is_new_user":    false,
			"is_linked":      true,
		})
	}
}
