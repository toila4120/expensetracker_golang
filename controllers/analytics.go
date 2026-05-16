package controllers

import (
	"expensetracker/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SearchTransactions tìm kiếm giao dịch theo từ khóa
func SearchTransactions(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)
		keyword := ctx.Query("q")

		if keyword == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập từ khóa tìm kiếm"})
			return
		}

		var transactions []models.Transaction
		searchPattern := "%" + keyword + "%"

		db.Where("user_id = ? AND (LOWER(note) LIKE LOWER(?) OR LOWER(category) LIKE LOWER(?))",
			userID, searchPattern, searchPattern).
			Order("date desc").
			Find(&transactions)

		ctx.JSON(http.StatusOK, gin.H{
			"keyword": keyword,
			"count":   len(transactions),
			"data":    transactions,
		})
	}
}

// GetCategoryStats trả về thống kê chi tiêu theo từng danh mục
func GetCategoryStats(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		// Mặc định lấy tháng hiện tại
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

		type CategoryStat struct {
			Category   string  `json:"category"`
			Total      int64   `json:"total"`
			Count      int64   `json:"count"`
			Percentage float64 `json:"percentage"`
		}

		// Tổng chi tiêu
		var totalExpense int64
		db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ? AND date BETWEEN ? AND ?",
				userID, "expense", startOfMonth, endOfMonth).
			Select("COALESCE(SUM(amount), 0)").Scan(&totalExpense)

		// Chi tiết theo danh mục
		type RawStat struct {
			Category string
			Total    int64
			Count    int64
		}

		var rawStats []RawStat
		db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ? AND date BETWEEN ? AND ?",
				userID, "expense", startOfMonth, endOfMonth).
			Select("category, SUM(amount) as total, COUNT(*) as count").
			Group("category").
			Order("total desc").
			Scan(&rawStats)

		var stats []CategoryStat
		for _, r := range rawStats {
			pct := float64(0)
			if totalExpense > 0 {
				pct = float64(r.Total) / float64(totalExpense) * 100
			}
			stats = append(stats, CategoryStat{
				Category:   r.Category,
				Total:      r.Total,
				Count:      r.Count,
				Percentage: pct,
			})
		}

		// Tổng thu nhập tháng này
		var totalIncome int64
		db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ? AND date BETWEEN ? AND ?",
				userID, "income", startOfMonth, endOfMonth).
			Select("COALESCE(SUM(amount), 0)").Scan(&totalIncome)

		ctx.JSON(http.StatusOK, gin.H{
			"month":         int(now.Month()),
			"year":          now.Year(),
			"total_expense": totalExpense,
			"total_income":  totalIncome,
			"categories":    stats,
		})
	}
}
