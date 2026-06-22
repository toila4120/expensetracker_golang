package controllers

import (
	"expensetracker/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateWalletInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Balance     int    `json:"balance"`
}

type InviteMemberInput struct {
	Email string `json:"email" binding:"required,email"`
}

func CreateWallet(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input CreateWalletInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		userID := c.MustGet("currentUserID").(uint)

		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Người dùng không tồn tại"})
			return
		}

		newWallet := models.Wallet{
			Name:        input.Name,
			Description: input.Description,
			Balance:     input.Balance,
			CreatedBy:   userID,
			Members:     []models.User{user},
		}

		if err := db.Create(&newWallet).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo ví, vui lòng thử lại"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Đã tạo ví thành công",
			"data":    newWallet,
		})
	}
}

func GetWallets(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)

		var wallets []models.Wallet
		// Tìm ví mà người dùng là thành viên
		err := db.Model(&models.Wallet{}).
			Joins("JOIN wallet_members ON wallet_members.wallet_id = wallets.id").
			Where("wallet_members.user_id = ?", userID).
			Preload("Members").
			Find(&wallets).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách ví"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": wallets,
		})
	}
}

func GetWalletDetails(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		walletIDStr := c.Param("id")
		walletID, err := strconv.ParseUint(walletIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID ví không hợp lệ"})
			return
		}

		var wallet models.Wallet
		if err := db.Preload("Members").First(&wallet, walletID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy ví"})
			return
		}

		// Xác thực thành viên
		isMember := false
		for _, member := range wallet.Members {
			if member.ID == userID {
				isMember = true
				break
			}
		}

		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền truy cập ví này"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": wallet,
		})
	}
}

func InviteMember(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		walletIDStr := c.Param("id")
		walletID, err := strconv.ParseUint(walletIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID ví không hợp lệ"})
			return
		}

		var input InviteMemberInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email không hợp lệ"})
			return
		}

		var wallet models.Wallet
		if err := db.Preload("Members").First(&wallet, walletID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy ví"})
			return
		}

		// Chỉ có chủ sở hữu ví mới có quyền mời
		if wallet.CreatedBy != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ chủ sở hữu ví mới có quyền mời thành viên"})
			return
		}

		var invitee models.User
		if err := db.Where("email = ?", input.Email).First(&invitee).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng với email này"})
			return
		}

		// Kiểm tra xem đã là thành viên chưa
		for _, member := range wallet.Members {
			if member.ID == invitee.ID {
				c.JSON(http.StatusConflict, gin.H{"error": "Người dùng đã là thành viên của ví này"})
				return
			}
		}

		// Thêm thành viên
		err = db.Model(&wallet).Association("Members").Append(&invitee)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm thành viên"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Đã mời thành viên thành công",
		})
	}
}
