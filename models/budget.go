package models

import "gorm.io/gorm"

type Budget struct {
	ID       int            `gorm:"primaryKey;autoIncrement"`
	UserID   uint           `gorm:"not null"`
	Category string         `gorm:"not null"`
	Amount   int            `gorm:"not null"` // Hạn mức ngân sách
	Month    int            `gorm:"not null"` // Tháng (1-12)
	Year     int            `gorm:"not null"` // Năm
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
