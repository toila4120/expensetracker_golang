// File: controllers/transaction.go
// Mục tiêu: Xử lý các nghiệp vụ (CRUD) liên quan đến dòng tiền.
//
// Nhiệm vụ của bạn:
// 1. CreateTransaction: Nhận dữ liệu thu chi, lấy userID từ Context (do middleware truyền vào), gán cho model Transaction và tạo mới trong DB.
// 2. GetTransactions:
//    - Lấy danh sách giao dịch CHỈ của user đang đăng nhập (where user_id = ?).
//    - Xử lý phân trang (Pagination): Nhận query params `page` và `limit` để chia nhỏ dữ liệu trả về.
//    - Xử lý lọc (Filter): Nhận query params `start_date`, `end_date` (để lấy theo khoảng thời gian) và `category`.
// 3. UpdateTransaction: Nhận ID từ URL, tìm giao dịch, xác minh giao dịch đó thuộc về user đang đăng nhập rồi tiến hành cập nhật.
// 4. DeleteTransaction: Nhận ID từ URL, xác minh quyền sở hữu, và xóa giao dịch.
//
// Kiến thức cần học:
// - Cách lấy query parameters (c.Query) và Path parameters (c.Param).
// - Truy vấn GORM nâng cao: Offset, Limit (sử dụng cho tính năng phân trang).
// - Dùng Where nâng cao (>, <, BETWEEN) để lọc theo khoảng thời gian ngày tháng.

package controllers

import (
	"expensetracker/models"
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
		}

		if err := db.Create(&newTransaction).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể tạo giao dịch, vui lòng thử lại sau",
			})
			return
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
		query := db.Model(&models.Transaction{}).Where("user_id = ?", userID)
		category := ctx.Query("category")
		if category != "" {
			query = query.Where("category = ?", category)
		}
		startDate := ctx.Query("start_date")
		endDate := ctx.Query("end_date")
		if startDate != "" && endDate != "" {
			query = query.Where("date BETWEEN ? AND ?", startDate+" 00:00:00", endDate+" 23:59:59")
		}
		page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
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
		transaction.Amount = input.Amount
		transaction.Category = input.Category
		transaction.Type = input.Type
		transaction.Note = input.Note
		if !input.Date.IsZero() {
			transaction.Date = input.Date
		}

		if err := db.Save(&transaction).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể cập nhật giao dịch, vui lòng thử lại",
			})
			return
		}
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
		if err := db.Delete(&transaction).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Lỗi hệ thống, không thể xóa giao dịch lúc này",
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Xóa giao dịch thành công",
		})
	}
}
