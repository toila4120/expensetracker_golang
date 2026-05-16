package controllers

import (
	"expensetracker/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetDashboard trả về tổng hợp tài chính cho user hiện tại
func GetDashboard(db *gorm.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := ctx.MustGet("currentUserID").(uint)

		var totalIncome int64
		var totalExpense int64
		var transactionCount int64

		db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ?", userID, "income").
			Select("COALESCE(SUM(amount), 0)").Scan(&totalIncome)

		db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ?", userID, "expense").
			Select("COALESCE(SUM(amount), 0)").Scan(&totalExpense)

		db.Model(&models.Transaction{}).
			Where("user_id = ?", userID).
			Count(&transactionCount)

		// Thống kê tháng hiện tại
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

		var monthlyIncome int64
		var monthlyExpense int64

		db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ? AND date BETWEEN ? AND ?", userID, "income", startOfMonth, endOfMonth).
			Select("COALESCE(SUM(amount), 0)").Scan(&monthlyIncome)

		db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ? AND date BETWEEN ? AND ?", userID, "expense", startOfMonth, endOfMonth).
			Select("COALESCE(SUM(amount), 0)").Scan(&monthlyExpense)

		ctx.JSON(http.StatusOK, gin.H{
			"total_income":     totalIncome,
			"total_expense":    totalExpense,
			"balance":          totalIncome - totalExpense,
			"transaction_count": transactionCount,
			"monthly_income":   monthlyIncome,
			"monthly_expense":  monthlyExpense,
			"monthly_balance":  monthlyIncome - monthlyExpense,
		})
	}
}
