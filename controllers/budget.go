package controllers

import (
	"expensetracker/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateBudgetInput struct {
	Category string `json:"category" binding:"required"`
	Amount   int    `json:"amount" binding:"required,gt=0"`
	Month    int    `json:"month"`
	Year     int    `json:"year"`
}

// CreateBudget tạo ngân sách cho một danh mục trong tháng
func CreateBudget(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		var input CreateBudgetInput
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		if input.Month == 0 {
			input.Month = int(time.Now().Month())
		}
		if input.Year == 0 {
			input.Year = time.Now().Year()
		}

		// Kiểm tra đã có budget cho category + tháng chưa
		var existing models.Budget
		if err := db.Where("user_id = ? AND category = ? AND month = ? AND year = ?",
			userID, input.Category, input.Month, input.Year).First(&existing).Error; err == nil {
			// Cập nhật nếu đã tồn tại
			existing.Amount = input.Amount
			db.Save(&existing)
			ctx.JSON(http.StatusOK, gin.H{
				"message": "Đã cập nhật ngân sách",
				"data":    existing,
			})
			return
		}

		budget := models.Budget{
			UserID:   userID,
			Category: input.Category,
			Amount:   input.Amount,
			Month:    input.Month,
			Year:     input.Year,
		}

		if err := db.Create(&budget).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo ngân sách"})
			return
		}

		ctx.JSON(http.StatusCreated, gin.H{
			"message": "Tạo ngân sách thành công",
			"data":    budget,
		})
	}
}

// GetBudgets lấy danh sách ngân sách theo tháng/năm
func GetBudgets(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		month, _ := strconv.Atoi(ctx.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
		year, _ := strconv.Atoi(ctx.DefaultQuery("year", strconv.Itoa(time.Now().Year())))

		var budgets []models.Budget
		db.Where("user_id = ? AND month = ? AND year = ?", userID, month, year).Find(&budgets)

		// Lấy tổng chi tiêu thực tế cho mỗi danh mục
		startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

		type BudgetStatus struct {
			Category string `json:"category"`
			Limit    int    `json:"limit"`
			Spent    int    `json:"spent"`
			Remaining int   `json:"remaining"`
			Percent  float64 `json:"percent"`
		}

		var result []BudgetStatus
		for _, b := range budgets {
			var spent int64
			db.Model(&models.Transaction{}).
				Where("user_id = ? AND category = ? AND type = ? AND date BETWEEN ? AND ?",
					userID, b.Category, "expense", startOfMonth, endOfMonth).
				Select("COALESCE(SUM(amount), 0)").Scan(&spent)

			pct := float64(0)
			if b.Amount > 0 {
				pct = float64(spent) / float64(b.Amount) * 100
			}

			result = append(result, BudgetStatus{
				Category:  b.Category,
				Limit:     b.Amount,
				Spent:     int(spent),
				Remaining: b.Amount - int(spent),
				Percent:   pct,
			})
		}

		ctx.JSON(http.StatusOK, gin.H{
			"month": month,
			"year":  year,
			"data":  result,
		})
	}
}

// DeleteBudget xóa ngân sách
func DeleteBudget(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)
		budgetID := ctx.Param("id")

		var budget models.Budget
		if err := db.Where("id = ? AND user_id = ?", budgetID, userID).First(&budget).Error; err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy ngân sách"})
			return
		}

		db.Delete(&budget)
		ctx.JSON(http.StatusOK, gin.H{"message": "Xóa ngân sách thành công"})
	}
}
