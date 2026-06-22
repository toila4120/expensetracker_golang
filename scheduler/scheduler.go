package scheduler

import (
	"expensetracker/models"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type Scheduler struct {
	cron *cron.Cron
	db   *gorm.DB
}

func New(db *gorm.DB) *Scheduler {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	return &Scheduler{
		cron: cron.New(cron.WithLocation(loc)),
		db:   db,
	}
}

func (s *Scheduler) Start() {
	s.cron.AddFunc("5 0 * * *", s.processRecurringTransactions)
	s.cron.Start()
	log.Println("⏰ Scheduler started - Recurring transactions will be processed daily at 00:05")
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("⏰ Scheduler stopped")
}

func (s *Scheduler) processRecurringTransactions() {
	now := time.Now()
	today := now.Day()
	currentMonth := int(now.Month())
	currentYear := now.Year()

	// Lấy tất cả giao dịch định kỳ đang active
	var recurrings []models.RecurringTransaction
	if err := s.db.Where("is_active = ?", true).Find(&recurrings).Error; err != nil {
		log.Println("❌ Scheduler: Lỗi khi lấy giao dịch định kỳ:", err)
		return
	}

	if len(recurrings) == 0 {
		log.Println("⏰ Scheduler: Không có giao dịch định kỳ nào cần xử lý")
		return
	}

	log.Printf("⏰ Scheduler: Tìm thấy %d giao dịch định kỳ active, kiểm tra ngày %d/%d/%d\n",
		len(recurrings), today, currentMonth, currentYear)

	createdCount := 0
	for _, recurring := range recurrings {
		// Xử lý trường hợp DayOfMonth lớn hơn số ngày trong tháng
		// Ví dụ: DayOfMonth=31 nhưng tháng 2 chỉ có 28 ngày -> tạo ngày cuối tháng
		targetDay := recurring.DayOfMonth
		lastDayOfMonth := time.Date(currentYear, time.Month(currentMonth+1), 0, 0, 0, 0, 0, time.Local).Day()
		if targetDay > lastDayOfMonth {
			targetDay = lastDayOfMonth
		}

		if today != targetDay {
			continue
		}

		// Kiểm tra đã tạo transaction cho ngày này chưa (tránh duplicate)
		startOfDay := time.Date(currentYear, time.Month(currentMonth), today, 0, 0, 0, 0, time.Local)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		var existing models.Transaction
		result := s.db.Where(
			"user_id = ? AND category = ? AND type = ? AND amount = ? AND note = ? AND date >= ? AND date < ?",
			recurring.UserID, recurring.Category, recurring.Type, recurring.Amount, recurring.Note,
			startOfDay, endOfDay,
		).First(&existing)

		if result.Error == nil {
			// Đã tồn tại transaction cho ngày này, bỏ qua
			log.Printf("⏰ Scheduler: Bỏ qua recurring #%d (user %d) - đã có transaction cho ngày %d/%d/%d\n",
				recurring.ID, recurring.UserID, today, currentMonth, currentYear)
			continue
		}

		// Tạo transaction mới
		transaction := models.Transaction{
			UserID:   recurring.UserID,
			Type:     recurring.Type,
			Amount:   recurring.Amount,
			Category: recurring.Category,
			Note:     recurring.Note,
			Date:     time.Date(currentYear, time.Month(currentMonth), today, 0, 0, 0, 0, time.Local),
		}

		if err := s.db.Create(&transaction).Error; err != nil {
			log.Printf("❌ Scheduler: Lỗi khi tạo transaction cho recurring #%d (user %d): %v\n",
				recurring.ID, recurring.UserID, err)
			continue
		}

		log.Printf("✅ Scheduler: Đã tạo transaction từ recurring #%d (user %d) - %s %d VND - %s\n",
			recurring.ID, recurring.UserID, recurring.Type, recurring.Amount, recurring.Category)
		createdCount++
	}

	log.Printf("⏰ Scheduler: Hoàn thành - Đã tạo %d/%d giao dịch mới\n", createdCount, len(recurrings))
}
