package scheduler

import (
	"expensetracker/models"
	"expensetracker/services"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type Scheduler struct {
	cron    *cron.Cron
	db      *gorm.DB
	notifSvc *services.NotificationService
}

func New(db *gorm.DB, notifSvc *services.NotificationService) *Scheduler {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	return &Scheduler{
		cron:    cron.New(cron.WithLocation(loc)),
		db:      db,
		notifSvc: notifSvc,
	}
}

func (s *Scheduler) Start() {
	s.catchUp()

	s.cron.AddFunc("5 0 * * *", s.processRecurringTransactions)
	s.cron.AddFunc("0 * * * *", s.sendDebtReminders)
	s.cron.Start()
	log.Println("⏰ Scheduler started - Recurring at 00:05, Debt reminders every hour")
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

		// Gửi thông báo cho user
		if s.notifSvc != nil {
			s.notifSvc.CreateAndDispatch(
				recurring.UserID,
				"recurring_created",
				"Giao dịch định kỳ",
				fmt.Sprintf("Đã tạo giao dịch tự động: %s %d VND - %s",
					recurring.Type, recurring.Amount, recurring.Category),
				nil,
				false, "", "", "",
			)
		}
	}

	if createdCount > 0 {
		log.Printf("⏰ Scheduler: Hoàn thành ngày %d/%d/%d - Đã tạo %d/%d giao dịch mới\n",
			day, month, year, createdCount, len(recurrings))
	}
}

type debtSummary struct {
	GroupMemberID uint   `json:"group_member_id"`
	UserID        *uint  `json:"user_id"`
	TotalOwed     int    `json:"total_owed"`
	GroupName     string `json:"group_name"`
	GroupID       uint   `json:"group_id"`
}

type debtBill struct {
	SharedBillID uint   `json:"shared_bill_id"`
	Amount       int    `json:"amount"`
	Description  string `json:"description"`
}

func (s *Scheduler) sendDebtReminders() {
	log.Println("⏰ Scheduler: Bắt đầu gửi nhắc nhở nợ...")
	today := time.Now().Truncate(24 * time.Hour)
	currentHour := time.Now().Hour()

	type debtSummary struct {
		GroupMemberID uint   `json:"group_member_id"`
		UserID        *uint  `json:"user_id"`
		TotalOwed     int    `json:"total_owed"`
		GroupName     string `json:"group_name"`
		GroupID       uint   `json:"group_id"`
	}

	type debtBill struct {
		SharedBillID uint   `json:"shared_bill_id"`
		Amount       int    `json:"amount"`
		Description  string `json:"description"`
	}

	// Quét theo bill: remind_auto=true và remind_hour khớp giờ hiện tại
	var summaries []debtSummary
	err := s.db.Raw(`
		SELECT bs.group_member_id, gm.user_id, SUM(bs.amount) AS total_owed,
		       g.name AS group_name, g.id AS group_id
		FROM bill_splits bs
		JOIN shared_bills sb ON sb.id = bs.shared_bill_id
		JOIN groups g ON g.id = sb.group_id
		JOIN group_members gm ON gm.id = bs.group_member_id
		WHERE bs.is_settled = false
		  AND sb.payer_id != bs.group_member_id
		  AND sb.remind_auto = true
		  AND sb.remind_hour = ?
		  AND bs.deleted_at IS NULL
		  AND sb.deleted_at IS NULL
		  AND g.deleted_at IS NULL
		GROUP BY bs.group_member_id, gm.user_id, g.name, g.id
	`, currentHour).Scan(&summaries).Error

	if err != nil {
		log.Println("❌ Scheduler: Lỗi khi lấy danh sách nợ:", err)
		return
	}

	if len(summaries) == 0 {
		log.Printf("⏰ Scheduler: Không có khoản nợ nào cần nhắc lúc %dh\n", currentHour)
		return
	}

	sentCount := 0
	for _, debt := range summaries {
		if debt.UserID == nil {
			continue
		}

		var todayReminders int64
		s.db.Model(&models.DebtReminder{}).
			Where("group_id = ? AND to_member_id = ? AND reminder_type = ? AND sent_at >= ?",
				debt.GroupID, debt.GroupMemberID, "auto", today).
			Count(&todayReminders)
		if todayReminders > 0 {
			continue
		}

		var bills []debtBill
		s.db.Raw(`
			SELECT sb.id AS shared_bill_id, bs.amount, sb.description
			FROM bill_splits bs
			JOIN shared_bills sb ON sb.id = bs.shared_bill_id
			WHERE sb.group_id = ?
			  AND bs.group_member_id = ?
			  AND bs.is_settled = false
			  AND sb.payer_id != bs.group_member_id
			  AND sb.remind_auto = true
			  AND sb.remind_hour = ?
			  AND bs.deleted_at IS NULL
			  AND sb.deleted_at IS NULL
		`, debt.GroupID, debt.GroupMemberID, currentHour).Scan(&bills)

		billDescs := ""
		for i, b := range bills {
			if i < 3 {
				if i > 0 {
					billDescs += ", "
				}
				billDescs += b.Description
			}
		}
		if len(bills) > 3 {
			billDescs += fmt.Sprintf(" và %d hóa đơn khác", len(bills)-3)
		}

		for _, b := range bills {
			reminder := models.DebtReminder{
				GroupID:      debt.GroupID,
				SharedBillID: b.SharedBillID,
				ToMemberID:   debt.GroupMemberID,
				ReminderType: "auto",
			}
			s.db.Create(&reminder)
		}

		if s.notifSvc != nil {
			var user models.User
			s.db.First(&user, *debt.UserID)
			if user.Email != "" {
				s.notifSvc.CreateAndDispatch(
					*debt.UserID,
					"debt_reminder",
					"Nhắc nhở thanh toán",
					fmt.Sprintf("Bạn nợ %d VND trong nhóm \"%s\" từ các hóa đơn: %s",
						debt.TotalOwed, debt.GroupName, billDescs),
					nil,
					true,
					user.Email,
					fmt.Sprintf("Nhắc nhở: Bạn có khoản nợ trong nhóm %s", debt.GroupName),
					fmt.Sprintf("Bạn đang nợ %d VND trong nhóm \"%s\" từ các hóa đơn: %s. Hãy thanh toán sớm nhé!",
						debt.TotalOwed, debt.GroupName, billDescs),
				)
			}
		}
		sentCount++
	}

	log.Printf("⏰ Scheduler: Hoàn thành nhắc nợ - Đã gửi %d nhắc nhở\n", sentCount)
}
