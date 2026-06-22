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
	// Bắt补齐 các ngày đã bỏ lỡ khi server vừa start
	s.catchUp()

	s.cron.AddFunc("5 0 * * *", s.processRecurringTransactions)
	s.cron.Start()
	log.Println("⏰ Scheduler started - Recurring transactions will be processed daily at 00:05")
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("⏰ Scheduler stopped")
}

// catchUp kiểm tra và xử lý các ngày bị bỏ lỡ từ lần chạy cuối cùng
func (s *Scheduler) catchUp() {
	lastRun := s.getLastRun()
	today := time.Now().Truncate(24 * time.Hour)

	if lastRun.IsZero() {
		// Lần đầu chạy, chỉ xử lý hôm nay
		log.Println("⏰ Scheduler: Chạy lần đầu, xử lý giao dịch hôm nay")
		s.processRecurringTransactionsForDate(today)
		s.updateLastRun(today)
		return
	}

	// Lấy ngày cuối cùng đã chạy (bỏ phần giờ)
	lastRunDay := lastRun.Truncate(24 * time.Hour)

	if !today.After(lastRunDay) {
		log.Println("⏰ Scheduler: Đã xử lý hôm nay rồi, bỏ qua")
		return
	}

	// Iterates từ ngày hôm sau lastRun đến hôm nay
	missedDays := 0
	for d := lastRunDay.AddDate(0, 0, 1); !d.After(today); d = d.AddDate(0, 0, 1) {
		log.Printf("⏰ Scheduler: Xử lý bù cho ngày %s\n", d.Format("02/01/2006"))
		s.processRecurringTransactionsForDate(d)
		missedDays++
	}

	s.updateLastRun(today)
	log.Printf("⏰ Scheduler: Đã xử lý bù %d ngày bị bỏ lỡ\n", missedDays)
}

// getLastRun lấy thời gian chạy cuối cùng từ DB
func (s *Scheduler) getLastRun() time.Time {
	var logEntry models.SchedulerLog
	// Lấy bản ghi mới nhất
	result := s.db.Order("id DESC").First(&logEntry)
	if result.Error != nil {
		return time.Time{} // Zero time nếu chưa có bản ghi nào
	}
	return logEntry.LastRunAt
}

// updateLastRun cập nhật thời gian chạy vào DB
func (s *Scheduler) updateLastRun(t time.Time) {
	logEntry := models.SchedulerLog{LastRunAt: t}
	s.db.Create(&logEntry)
}

// processRecurringTransactions được gọi bởi cron job (xử lý hôm nay)
func (s *Scheduler) processRecurringTransactions() {
	today := time.Now().Truncate(24 * time.Hour)
	s.processRecurringTransactionsForDate(today)
	s.updateLastRun(today)
}

// processRecurringTransactionsForDate xử lý recurring transactions cho một ngày cụ thể
func (s *Scheduler) processRecurringTransactionsForDate(targetDate time.Time) {
	day := targetDate.Day()
	month := int(targetDate.Month())
	year := targetDate.Year()

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
		len(recurrings), day, month, year)

	createdCount := 0
	for _, recurring := range recurrings {
		// Xử lý DayOfMonth > số ngày trong tháng (VD: DayOfMonth=31 ở tháng 2 → ngày 28/29)
		targetDay := recurring.DayOfMonth
		lastDayOfMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.Local).Day()
		if targetDay > lastDayOfMonth {
			targetDay = lastDayOfMonth
		}

		if day != targetDay {
			continue
		}

		// Kiểm tra đã tạo transaction cho ngày này chưa (tránh duplicate)
		startOfDay := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		var existing models.Transaction
		result := s.db.Where(
			"user_id = ? AND category = ? AND type = ? AND amount = ? AND note = ? AND date >= ? AND date < ?",
			recurring.UserID, recurring.Category, recurring.Type, recurring.Amount, recurring.Note,
			startOfDay, endOfDay,
		).First(&existing)

		if result.Error == nil {
			log.Printf("⏰ Scheduler: Bỏ qua recurring #%d (user %d) - đã có transaction cho ngày %d/%d/%d\n",
				recurring.ID, recurring.UserID, day, month, year)
			continue
		}

		// Tạo transaction mới
		transaction := models.Transaction{
			UserID:   recurring.UserID,
			Type:     recurring.Type,
			Amount:   recurring.Amount,
			Category: recurring.Category,
			Note:     recurring.Note,
			Date:     time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local),
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

	if createdCount > 0 {
		log.Printf("⏰ Scheduler: Hoàn thành ngày %d/%d/%d - Đã tạo %d/%d giao dịch mới\n",
			day, month, year, createdCount, len(recurrings))
	}
}
