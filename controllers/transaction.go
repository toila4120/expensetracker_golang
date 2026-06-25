// File: controllers/transaction.go
package controllers

import (
	"expensetracker/models"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateTransactionInput struct {
	Amount      int       `json:"amount" binding:"required,gt=0"`
	Category    string    `json:"category" binding:"required"`
	Type        string    `json:"type" binding:"required,oneof=income expense"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	WalletID    *uint     `json:"wallet_id"`
}

func CreateTransaction(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input CreateTransactionInput

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		userID := c.MustGet("currentUserID").(uint)

		var wallet models.Wallet
		if input.WalletID != nil {
			// Xác thực xem người dùng có là thành viên của ví không
			var count int64
			db.Table("wallet_members").Where("wallet_id = ? AND user_id = ?", *input.WalletID, userID).Count(&count)
			if count == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của ví này"})
				return
			}
			if err := db.First(&wallet, *input.WalletID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Ví không tồn tại"})
				return
			}
		}

		transactionDate := input.Date
		if transactionDate.IsZero() {
			transactionDate = time.Now()
		}

		newTransaction := models.Transaction{
			Amount:   input.Amount,
			Category: input.Category,
			Type:     input.Type,
			Note:     input.Description,
			Date:     transactionDate,
			UserID:   userID,
			WalletID: input.WalletID,
		}

		tx := db.Begin()
		if err := tx.Create(&newTransaction).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể tạo giao dịch, vui lòng thử lại sau",
			})
			return
		}

		if input.WalletID != nil {
			if newTransaction.Type == "income" {
				wallet.Balance += newTransaction.Amount
			} else {
				wallet.Balance -= newTransaction.Amount
			}
			if err := tx.Save(&wallet).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Không thể cập nhật số dư ví",
				})
				return
			}
		}
		tx.Commit()

		// Kiểm tra ngân sách và gửi thông báo nếu vượt ngưỡng
		if newTransaction.Type == "expense" && NotifSvc != nil {
			go checkBudgetNotification(db, userID, newTransaction.Category, transactionDate)
		}

		// Tự động phân bổ income vào financial goals nếu có bật auto_allocate
		if newTransaction.Type == "income" {
			go AutoAllocateToGoals(db, userID, newTransaction.Amount, newTransaction.WalletID)
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Đã thêm giao dịch thành công",
			"data":    newTransaction,
		})
	}
}

func GetAllTransaction(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		// Người dùng có thể thấy giao dịch của mình HOẶC giao dịch thuộc ví chung mà họ làm thành viên
		query := db.Model(&models.Transaction{}).
			Where("transactions.user_id = ? OR transactions.wallet_id IN (SELECT wallet_id FROM wallet_members WHERE user_id = ?)", userID, userID)

		walletIDStr := ctx.Query("wallet_id")
		if walletIDStr != "" {
			walletID, err := strconv.ParseUint(walletIDStr, 10, 32)
			if err == nil {
				// Xác minh xem có phải thành viên của ví đó không
				var count int64
				db.Table("wallet_members").Where("wallet_id = ? AND user_id = ?", walletID, userID).Count(&count)
				if count == 0 {
					ctx.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của ví này"})
					return
				}
				query = query.Where("transactions.wallet_id = ?", walletID)
			}
		}

		category := ctx.Query("category")
		if category != "" {
			query = query.Where("transactions.category = ?", category)
		}
		startDate := ctx.Query("start_date")
		endDate := ctx.Query("end_date")
		if startDate != "" && endDate != "" {
			query = query.Where("transactions.date BETWEEN ? AND ?", startDate+" 00:00:00", endDate+" 23:59:59")
		}
		page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
		offset := (page - 1) * limit
		var transactions []models.Transaction
		if err := query.Offset(offset).Limit(limit).Order("date desc").Find(&transactions).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách giao dịch"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"page":  page,
			"limit": limit,
			"count": len(transactions),
			"data":  transactions,
		})
	}
}

type UpdateTransactionInput struct {
	Amount   int       `json:"amount" binding:"required,gt=0"`
	Category string    `json:"category" binding:"required"`
	Type     string    `json:"type" binding:"required,oneof=income expense"`
	Note     string    `json:"description"`
	Date     time.Time `json:"date"`
	WalletID *uint     `json:"wallet_id"`
}

func UpdateTransaction(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)
		var transaction models.Transaction
		transactionID := ctx.Param("id")
		if err := db.Where("id = ? AND user_id = ?", transactionID, userID).First(&transaction).Error; err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Không tìm thấy giao dịch hoặc bạn không có quyền chỉnh sửa",
			})
			return
		}
		var input UpdateTransactionInput
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		tx := db.Begin()

		// Hoàn tác ảnh hưởng ví cũ
		if transaction.WalletID != nil {
			var wallet models.Wallet
			if err := tx.First(&wallet, *transaction.WalletID).Error; err == nil {
				if transaction.Type == "income" {
					wallet.Balance -= transaction.Amount
				} else {
					wallet.Balance += transaction.Amount
				}
				tx.Save(&wallet)
			}
		}

		// Gán giá trị mới
		transaction.Amount = input.Amount
		transaction.Category = input.Category
		transaction.Type = input.Type
		transaction.Note = input.Note
		transaction.WalletID = input.WalletID
		if !input.Date.IsZero() {
			transaction.Date = input.Date
		}

		// Áp dụng ảnh hưởng ví mới
		if transaction.WalletID != nil {
			var count int64
			tx.Table("wallet_members").Where("wallet_id = ? AND user_id = ?", *transaction.WalletID, userID).Count(&count)
			if count == 0 {
				tx.Rollback()
				ctx.JSON(http.StatusForbidden, gin.H{"error": "Bạn không phải thành viên của ví này"})
				return
			}

			var wallet models.Wallet
			if err := tx.First(&wallet, *transaction.WalletID).Error; err != nil {
				tx.Rollback()
				ctx.JSON(http.StatusNotFound, gin.H{"error": "Ví không tồn tại"})
				return
			}
			if transaction.Type == "income" {
				wallet.Balance += transaction.Amount
			} else {
				wallet.Balance -= transaction.Amount
			}
			tx.Save(&wallet)
		}

		if err := tx.Save(&transaction).Error; err != nil {
			tx.Rollback()
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể cập nhật giao dịch, vui lòng thử lại",
			})
			return
		}

		tx.Commit()

		ctx.JSON(http.StatusOK, gin.H{
			"message": "Cập nhật giao dịch thành công",
			"data":    transaction,
		})
	}
}

func DeleteTransaction(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)
		var transaction models.Transaction
		transactionID := ctx.Param("id")
		if err := db.Where("id = ? AND user_id = ?", transactionID, userID).First(&transaction).Error; err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Không tìm thấy giao dịch hoặc bạn không có quyền chỉnh sửa",
			})
			return
		}

		tx := db.Begin()

		// Hoàn tác ảnh hưởng ví
		if transaction.WalletID != nil {
			var wallet models.Wallet
			if err := tx.First(&wallet, *transaction.WalletID).Error; err == nil {
				if transaction.Type == "income" {
					wallet.Balance -= transaction.Amount
				} else {
					wallet.Balance += transaction.Amount
				}
				tx.Save(&wallet)
			}
		}

		if err := tx.Delete(&transaction).Error; err != nil {
			tx.Rollback()
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống, không thể xóa giao dịch lúc này",
			})
			return
		}

		tx.Commit()

		ctx.JSON(http.StatusOK, gin.H{
			"message": "Xóa giao dịch thành công",
		})
	}
}

// checkBudgetNotification kiểm tra ngân sách và gửi thông báo khi vượt ngưỡng
func checkBudgetNotification(db *gorm.DB, userID uint, category string, transactionDate time.Time) {
	month := int(transactionDate.Month())
	year := transactionDate.Year()

	log.Printf("checkBudgetNotification: userID=%d, category=%s, month=%d, year=%d", userID, category, month, year)

	var budget models.Budget
	if err := db.Where("user_id = ? AND category = ? AND month = ? AND year = ?",
		userID, category, month, year).First(&budget).Error; err != nil {
		log.Printf("checkBudgetNotification: No budget found for category %s", category)
		return // Không có ngân sách cho danh mục này
	}

	log.Printf("checkBudgetNotification: Found budget %d VND for category %s", budget.Amount, category)

	// Tính tổng chi tiêu trong tháng cho danh mục này
	var totalSpent int
	db.Model(&models.Transaction{}).
		Where("user_id = ? AND category = ? AND type = ? AND MONTH(date) = ? AND YEAR(date) = ?",
			userID, category, "expense", month, year).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalSpent)

	log.Printf("checkBudgetNotification: Total spent=%d, budget=%d, percentage=%d%%", 
		totalSpent, budget.Amount, totalSpent*100/budget.Amount)

	// Kiểm tra ngưỡng 80%
	if totalSpent > budget.Amount*80/100 && totalSpent <= budget.Amount {
		log.Printf("checkBudgetNotification: Sending budget_warning notification")
		NotifSvc.CreateAndDispatch(
			userID,
			"budget_warning",
			"Cảnh báo ngân sách",
			fmt.Sprintf("Chi tiêu danh mục %s đã vượt 80%% ngân sách (%d/%d VND)",
				category, totalSpent, budget.Amount),
			nil,
			false, "", "", "",
		)
	}

	// Kiểm tra vượt ngân sách
	if totalSpent > budget.Amount {
		log.Printf("checkBudgetNotification: Sending budget_exceeded notification")
		NotifSvc.CreateAndDispatch(
			userID,
			"budget_exceeded",
			"Vượt ngân sách",
			fmt.Sprintf("Chi tiêu danh mục %s đã vượt ngân sách (%d/%d VND)",
				category, totalSpent, budget.Amount),
			nil,
			true,
			NotifSvc.GetUserEmail(userID),
			fmt.Sprintf("Cảnh báo: Vượt ngân sách danh mục %s", category),
			fmt.Sprintf("Bạn đã chi tiêu %d VND cho danh mục %s, vượt quá ngân sách %d VND.",
				totalSpent, category, budget.Amount),
		)
	}
}
