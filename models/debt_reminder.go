package models

import "time"

type DebtReminder struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID      uint      `gorm:"not null;index" json:"group_id"`
	SharedBillID uint      `gorm:"not null;index" json:"shared_bill_id"`
	FromMemberID uint      `gorm:"not null" json:"from_member_id"`
	ToMemberID   uint      `gorm:"not null" json:"to_member_id"`
	ReminderType string    `gorm:"not null" json:"reminder_type"` // "manual" | "auto"
	SentAt       time.Time `gorm:"autoCreateTime" json:"sent_at"`
}
