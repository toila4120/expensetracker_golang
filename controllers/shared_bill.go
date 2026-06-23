package controllers

import (
	"expensetracker/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateBillInput struct {
	PayerID         uint              `json:"payer_id" binding:"required"`
	Amount          int               `json:"amount" binding:"required,gt=0"`
	Category        string            `json:"category" binding:"required,oneof=food transport shopping entertainment education health other"`
	Description     string            `json:"description"`
	SplitMethod     string            `json:"split_method" binding:"required,oneof=equal percentage custom"`
	TransactionDate time.Time         `json:"transaction_date"`
	Splits          []BillSplitInput  `json:"splits" binding:"required,min=1"`
}

type BillSplitInput struct {
	GroupMemberID uint `json:"group_member_id" binding:"required"`
	Amount        int  `json:"amount" binding:"required,gt=0"`
}

func CreateSharedBill(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			return
		}

		var input CreateBillInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ", "details": err.Error()})
			return
		}

		var group models.Group
		if err := db.First(&group, groupID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhóm"})
			return
		}

		var isMember int64
		db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, userID).Count(&isMember)
		if isMember == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của nhóm này"})
			return
		}

		var payerMember models.GroupMember
		if err := db.Where("id = ? AND group_id = ?", input.PayerID, group.ID).First(&payerMember).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Người trả tiền không phải thành viên của nhóm"})
			return
		}

		memberIDs := make(map[uint]bool)
		var members []models.GroupMember
		db.Where("group_id = ?", group.ID).Find(&members)
		for _, m := range members {
			memberIDs[m.ID] = true
		}

		for _, split := range input.Splits {
			if !memberIDs[split.GroupMemberID] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Thành viên chia tiền không phải thành viên của nhóm"})
				return
			}
		}

		seenMembers := make(map[uint]bool)
		totalSplit := 0
		for _, split := range input.Splits {
			if seenMembers[split.GroupMemberID] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Một thành viên chỉ được xuất hiện một lần trong danh sách chia tiền"})
				return
			}
			seenMembers[split.GroupMemberID] = true
			totalSplit += split.Amount
		}

		if totalSplit != input.Amount {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":        "Tổng tiền chia không khớp với tổng hóa đơn",
				"bill_amount":  input.Amount,
				"split_total":  totalSplit,
			})
			return
		}

		txDate := input.TransactionDate
		if txDate.IsZero() {
			txDate = time.Now()
		}

		tx := db.Begin()

		sharedBill := models.SharedBill{
			GroupID:         group.ID,
			PayerID:         payerMember.ID,
			CreatorID:       userID,
			Amount:          input.Amount,
			Category:        input.Category,
			Description:     input.Description,
			SplitMethod:     input.SplitMethod,
			TransactionDate: txDate,
		}
		if err := tx.Create(&sharedBill).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo hóa đơn"})
			return
		}

		for _, split := range input.Splits {
			billSplit := models.BillSplit{
				SharedBillID:  sharedBill.ID,
				GroupMemberID: split.GroupMemberID,
				Amount:        split.Amount,
				IsSettled:     false,
			}
			if err := tx.Create(&billSplit).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo chi tiết chia tiền"})
				return
			}
		}

		tx.Commit()

		c.JSON(http.StatusCreated, gin.H{
			"message": "Tạo hóa đơn thành công",
			"data":    sharedBill,
		})
	}
}

func GetSharedBills(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID nhóm không hợp lệ"})
			return
		}

		var isMember int64
		db.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&isMember)
		if isMember == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của nhóm này"})
			return
		}

		var bills []models.SharedBill
		if err := db.Where("group_id = ?", groupID).Order("transaction_date DESC").Find(&bills).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách hóa đơn"})
			return
		}

		type BillResponse struct {
			models.SharedBill
			PayerName    string            `json:"payer_name"`
			CreatorName  string            `json:"creator_name"`
			Splits       []models.BillSplit `json:"splits"`
		}

		var responses []BillResponse
		for _, bill := range bills {
			var payerMember models.GroupMember
			db.First(&payerMember, bill.PayerID)

			var creator models.User
			db.First(&creator, bill.CreatorID)

			var splits []models.BillSplit
			db.Where("shared_bill_id = ?", bill.ID).Find(&splits)

			payerName := ""
			if payerMember.UserID != nil {
				var user models.User
				db.First(&user, *payerMember.UserID)
				payerName = user.Username
			} else {
				payerName = payerMember.GuestName
			}

			responses = append(responses, BillResponse{
				SharedBill:  bill,
				PayerName:   payerName,
				CreatorName: creator.Username,
				Splits:      splits,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": responses})
	}
}
