package controllers

import (
	"expensetracker/models"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// getPersonalWallet lấy ví cá nhân của user (wallet chỉ có 1 member là chính họ)
func getPersonalWallet(db *gorm.DB, userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := db.Raw(`
		SELECT w.* FROM wallets w
	 INNER JOIN wallet_members wm ON wm.wallet_id = w.id
	 WHERE w.created_by = ? AND w.deleted_at IS NULL
	 GROUP BY w.id
	 HAVING COUNT(wm.user_id) = 1
	`, userID).Scan(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

type CreateGoalInput struct {
	Name           string     `json:"name" binding:"required"`
	TargetAmount   int        `json:"target_amount" binding:"required,gt=0"`
	Deadline       *time.Time `json:"deadline"`
	Category       string     `json:"category" binding:"required,oneof=savings travel emergency education investment"`
	Icon           string     `json:"icon"`
	AutoAllocate   bool       `json:"auto_allocate"`
	AllocatePercent int       `json:"allocate_percent" binding:"omitempty,max=100,min=0"`
}

// CreateGoal tạo mục tiêu tài chính mới
func CreateGoal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)

		var input CreateGoalInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		if input.AutoAllocate && (input.AllocatePercent <= 0 || input.AllocatePercent > 100) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Phần trăm phân bổ phải từ 1 đến 100 khi bật tự động phân bổ",
			})
			return
		}

		goal := models.FinancialGoal{
			UserID:          userID,
			Name:            input.Name,
			TargetAmount:    input.TargetAmount,
			CurrentAmount:   0,
			Deadline:        input.Deadline,
			Category:        input.Category,
			Icon:            input.Icon,
			AutoAllocate:    input.AutoAllocate,
			AllocatePercent: input.AllocatePercent,
			IsActive:        true,
		}

		if err := db.Create(&goal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo mục tiêu"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Tạo mục tiêu thành công",
			"data":    goal,
		})
	}
}

// GetGoals lấy danh sách mục tiêu với tiến độ
func GetGoals(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)

		var goals []models.FinancialGoal
		if err := db.Where("user_id = ?", userID).Order("created_at desc").Find(&goals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách mục tiêu"})
			return
		}

		type GoalResponse struct {
			models.FinancialGoal
			Progress       float64 `json:"progress"`        // % hoàn thành
			Remaining      int     `json:"remaining"`       // Số tiền còn thiếu
			DaysLeft       *int    `json:"days_left"`       // Số ngày còn lại (nil nếu không có deadline)
			IsOverBudget   bool    `json:"is_over_budget"`  // Đã vượt mục tiêu chưa
			IsExpired      bool    `json:"is_expired"`      // Đã hết hạn chưa
		}

		var responses []GoalResponse
		now := time.Now()

		for _, goal := range goals {
			progress := 0.0
			if goal.TargetAmount > 0 {
				progress = float64(goal.CurrentAmount) / float64(goal.TargetAmount) * 100
				progress = math.Min(progress, 100)
			}

			remaining := goal.TargetAmount - goal.CurrentAmount
			if remaining < 0 {
				remaining = 0
			}

			var daysLeft *int
			var isExpired bool
			if goal.Deadline != nil {
				days := int(time.Until(*goal.Deadline).Hours() / 24)
				daysLeft = &days
				isExpired = now.After(*goal.Deadline)
			}

			responses = append(responses, GoalResponse{
				FinancialGoal: goal,
				Progress:      math.Round(progress*100) / 100,
				Remaining:     remaining,
				DaysLeft:      daysLeft,
				IsOverBudget:  goal.CurrentAmount >= goal.TargetAmount,
				IsExpired:     isExpired,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"count": len(responses),
			"data":  responses,
		})
	}
}

// GetGoalDetails xem chi tiết mục tiêu
func GetGoalDetails(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		goalID := c.Param("id")

		var goal models.FinancialGoal
		if err := db.Where("id = ? AND user_id = ?", goalID, userID).First(&goal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mục tiêu"})
			return
		}

		progress := 0.0
		if goal.TargetAmount > 0 {
			progress = float64(goal.CurrentAmount) / float64(goal.TargetAmount) * 100
			progress = math.Min(progress, 100)
		}

		remaining := goal.TargetAmount - goal.CurrentAmount
		if remaining < 0 {
			remaining = 0
		}

		now := time.Now()
		var daysLeft *int
		var isExpired bool
		if goal.Deadline != nil {
			days := int(time.Until(*goal.Deadline).Hours() / 24)
			daysLeft = &days
			isExpired = now.After(*goal.Deadline)
		}

		// Tính toán pace (tốc độ tiết kiệm cần thiết mỗi ngày để đạt deadline)
		var dailyRequired *float64
		if goal.Deadline != nil && !isExpired && remaining > 0 {
			days := time.Until(*goal.Deadline).Hours() / 24
			if days > 0 {
				dr := float64(remaining) / days
				dailyRequired = &dr
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": goal,
			"progress": gin.H{
				"percentage":    math.Round(progress*100) / 100,
				"remaining":     remaining,
				"days_left":     daysLeft,
				"is_over_budget": goal.CurrentAmount >= goal.TargetAmount,
				"is_expired":    isExpired,
				"daily_required": dailyRequired,
			},
		})
	}
}

type UpdateGoalInput struct {
	Name            string     `json:"name"`
	TargetAmount    int        `json:"target_amount" binding:"omitempty,gt=0"`
	Deadline        *time.Time `json:"deadline"`
	Category        string     `json:"category" binding:"omitempty,oneof=savings travel emergency education investment"`
	Icon            string     `json:"icon"`
	AutoAllocate    *bool      `json:"auto_allocate"`
	AllocatePercent *int       `json:"allocate_percent" binding:"omitempty,max=100,min=0"`
	IsActive        *bool      `json:"is_active"`
}

// UpdateGoal cập nhật mục tiêu
func UpdateGoal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		goalID := c.Param("id")

		var goal models.FinancialGoal
		if err := db.Where("id = ? AND user_id = ?", goalID, userID).First(&goal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mục tiêu"})
			return
		}

		var input UpdateGoalInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		if input.Name != "" {
			goal.Name = input.Name
		}
		if input.TargetAmount > 0 {
			goal.TargetAmount = input.TargetAmount
		}
		if input.Deadline != nil {
			goal.Deadline = input.Deadline
		}
		if input.Category != "" {
			goal.Category = input.Category
		}
		if input.Icon != "" {
			goal.Icon = input.Icon
		}
		if input.AutoAllocate != nil {
			goal.AutoAllocate = *input.AutoAllocate
		}
		if input.AllocatePercent != nil {
			goal.AllocatePercent = *input.AllocatePercent
		}
		if input.IsActive != nil {
			goal.IsActive = *input.IsActive
		}

		if err := db.Save(&goal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật mục tiêu"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Cập nhật mục tiêu thành công",
			"data":    goal,
		})
	}
}

// DeleteGoal xóa mục tiêu (soft delete)
func DeleteGoal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		goalID := c.Param("id")

		var goal models.FinancialGoal
		if err := db.Where("id = ? AND user_id = ?", goalID, userID).First(&goal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mục tiêu"})
			return
		}

		if err := db.Delete(&goal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa mục tiêu"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Xóa mục tiêu thành công"})
	}
}

type AllocateInput struct {
	Amount int `json:"amount" binding:"required,gt=0"`
}

// AllocateToGoal phân bổ tiền vào mục tiêu từ ví cá nhân
func AllocateToGoal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		goalID := c.Param("id")

		var goal models.FinancialGoal
		if err := db.Where("id = ? AND user_id = ? AND is_active = ?", goalID, userID, true).First(&goal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mục tiêu hoặc mục tiêu đã tắt"})
			return
		}

		var input AllocateInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		// Lấy ví cá nhân
		wallet, err := getPersonalWallet(db, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy ví cá nhân"})
			return
		}

		// Kiểm tra số tiền không vượt quá goal còn thiếu
		goalRemaining := goal.TargetAmount - goal.CurrentAmount
		if input.Amount > goalRemaining {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":          "Số tiền vượt quá mục tiêu của quỹ",
				"goal_remaining": goalRemaining,
			})
			return
		}

		// Tạo transaction ghi nhận việc phân bổ từ ví vào goal
		transaction := models.Transaction{
			UserID:   userID,
			WalletID: &wallet.ID,
			Type:     "expense",
			Category: "goal_allocation",
			Amount:   input.Amount,
			Note:     fmt.Sprintf("Phân bổ vào mục tiêu: %s", goal.Name),
			Date:     time.Now(),
		}

		// Dùng DB transaction để đảm bảo atomic
		tx := db.Begin()

		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo giao dịch"})
			return
		}

		wallet.Balance -= input.Amount
		if err := tx.Save(wallet).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật số dư ví"})
			return
		}

		goal.CurrentAmount += input.Amount
		if err := tx.Save(&goal).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể phân bổ tiền vào mục tiêu"})
			return
		}

		tx.Commit()

		progress := 0.0
		if goal.TargetAmount > 0 {
			progress = float64(goal.CurrentAmount) / float64(goal.TargetAmount) * 100
		}

		// Check goal notifications
		go checkGoalNotifications(db, userID, &goal)

		c.JSON(http.StatusOK, gin.H{
			"message": "Phân bổ tiền vào mục tiêu thành công",
			"data": gin.H{
				"id":              goal.ID,
				"name":            goal.Name,
				"current_amount":  goal.CurrentAmount,
				"target_amount":   goal.TargetAmount,
				"progress":        math.Round(progress*100) / 100,
				"allocated_amount": input.Amount,
				"wallet_balance":  wallet.Balance,
			},
		})
	}
}

// WithdrawFromGoal rút tiền từ mục tiêu về ví cá nhân
func WithdrawFromGoal(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet("currentUserID").(uint)
		goalID := c.Param("id")

		var goal models.FinancialGoal
		if err := db.Where("id = ? AND user_id = ?", goalID, userID).First(&goal).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mục tiêu"})
			return
		}

		var input AllocateInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Dữ liệu không hợp lệ",
				"details": err.Error(),
			})
			return
		}

		if input.Amount > goal.CurrentAmount {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":          "Số tiền rút vượt quá số tiền hiện có trong mục tiêu",
				"current_amount": goal.CurrentAmount,
			})
			return
		}

		// Lấy ví cá nhân
		wallet, err := getPersonalWallet(db, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy ví cá nhân"})
			return
		}

		// Tạo transaction ghi nhận việc rút tiền từ goal về ví
		transaction := models.Transaction{
			UserID:   userID,
			WalletID: &wallet.ID,
			Type:     "income",
			Category: "goal_withdrawal",
			Amount:   input.Amount,
			Note:     fmt.Sprintf("Rút từ mục tiêu: %s", goal.Name),
			Date:     time.Now(),
		}

		// Dùng DB transaction để đảm bảo atomic
		tx := db.Begin()

		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo giao dịch"})
			return
		}

		wallet.Balance += input.Amount
		if err := tx.Save(wallet).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật số dư ví"})
			return
		}

		goal.CurrentAmount -= input.Amount
		if err := tx.Save(&goal).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể rút tiền từ mục tiêu"})
			return
		}

		tx.Commit()

		c.JSON(http.StatusOK, gin.H{
			"message": "Rút tiền từ mục tiêu thành công",
			"data": gin.H{
				"id":             goal.ID,
				"name":           goal.Name,
				"current_amount": goal.CurrentAmount,
				"target_amount":  goal.TargetAmount,
				"withdrawn":      input.Amount,
				"wallet_balance": wallet.Balance,
			},
		})
	}
}

// AutoAllocateToGoals phân bổ tự động từ income vào các goal có bật auto_allocate
// Gọi hàm này sau khi tạo income transaction
func AutoAllocateToGoals(db *gorm.DB, userID uint, incomeAmount int, walletID *uint) {
	var goals []models.FinancialGoal
	db.Where("user_id = ? AND auto_allocate = ? AND is_active = ?", userID, true, true).Find(&goals)

	if len(goals) == 0 {
		return
	}

	// Lấy ví cá nhân nếu có walletID
	var wallet *models.Wallet
	if walletID != nil {
		var w models.Wallet
		if err := db.First(&w, *walletID).Error; err == nil {
			wallet = &w
		}
	}
	if wallet == nil {
		var err error
		wallet, err = getPersonalWallet(db, userID)
		if err != nil {
			return
		}
	}

	remaining := incomeAmount
	for _, goal := range goals {
		if remaining <= 0 {
			break
		}

		// Tính số tiền phân bổ theo %
		allocateAmount := incomeAmount * goal.AllocatePercent / 100
		if allocateAmount <= 0 {
			continue
		}

		// Không phân bổ quá số tiền còn thiếu
		goalRemaining := goal.TargetAmount - goal.CurrentAmount
		if goalRemaining <= 0 {
			continue // Mục tiêu đã hoàn thành
		}
		if allocateAmount > goalRemaining {
			allocateAmount = goalRemaining
		}
		if allocateAmount > remaining {
			allocateAmount = remaining
		}

		// Tạo transaction ghi nhận việc phân bổ từ ví vào goal
		transaction := models.Transaction{
			UserID:   userID,
			WalletID: &wallet.ID,
			Type:     "expense",
			Category: "goal_allocation",
			Amount:   allocateAmount,
			Note:     fmt.Sprintf("Tự động phân bổ vào mục tiêu: %s", goal.Name),
			Date:     time.Now(),
		}

		tx := db.Begin()

		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			continue
		}

		wallet.Balance -= allocateAmount
		if err := tx.Save(wallet).Error; err != nil {
			tx.Rollback()
			continue
		}

		goal.CurrentAmount += allocateAmount
		if err := tx.Save(&goal).Error; err != nil {
			tx.Rollback()
			continue
		}

		tx.Commit()
		remaining -= allocateAmount

		// Check goal notifications after auto-allocation
		if NotifSvc != nil {
			go checkGoalNotifications(db, userID, &goal)
		}
	}
}

// checkGoalNotifications kiểm tra và gửi thông báo cho goal
func checkGoalNotifications(db *gorm.DB, userID uint, goal *models.FinancialGoal) {
	if NotifSvc == nil {
		return
	}

	// Goal completed
	if goal.CurrentAmount >= goal.TargetAmount {
		NotifSvc.CreateAndDispatch(
			userID,
			"goal_completed",
			"Mục tiêu hoàn thành!",
			fmt.Sprintf("Mục tiêu %s đã hoàn thành (%d/%d VND)",
				goal.Name, goal.CurrentAmount, goal.TargetAmount),
			nil,
			true,
			NotifSvc.GetUserEmail(userID),
			fmt.Sprintf("Chúc mừng! Mục tiêu %s đã hoàn thành", goal.Name),
			fmt.Sprintf("Mục tiêu tiết kiệm %s của bạn đã hoàn thành với số tiền %d VND.",
				goal.Name, goal.CurrentAmount),
		)
		return
	}

	// Goal deadline approaching
	if goal.Deadline != nil {
		daysLeft := int(time.Until(*goal.Deadline).Hours() / 24)
		if daysLeft >= 0 && daysLeft < 7 {
			NotifSvc.CreateAndDispatch(
				userID,
				"goal_deadline",
				"Mục tiêu sắp hết hạn",
				fmt.Sprintf("Mục tiêu %s còn %d ngày hết hạn (còn thiếu %d VND)",
					goal.Name, daysLeft, goal.TargetAmount-goal.CurrentAmount),
				nil,
				true,
				NotifSvc.GetUserEmail(userID),
				fmt.Sprintf("Cảnh báo: Mục tiêu %s sắp hết hạn", goal.Name),
				fmt.Sprintf("Mục tiêu %s của bạn sẽ hết hạn trong %d ngày. Bạn còn thiếu %d VND.",
					goal.Name, daysLeft, goal.TargetAmount-goal.CurrentAmount),
			)
		}
	}
}
