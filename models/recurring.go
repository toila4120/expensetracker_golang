package models

import "gorm.io/gorm"

type RecurringTransaction struct {
	ID        int            `gorm:"primaryKey;autoIncrement"`
	UserID    uint           `gorm:"not null"`
	Amount    int            `gorm:"not null"`
	Category  string         `gorm:"not null"`
	Type      string         `gorm:"not null"` // income or expense
	Note      string
	DayOfMonth int           `gorm:"not null"` // Ngày trong tháng (1-31)
	IsActive  bool           `gorm:"default:true"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
