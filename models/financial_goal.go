package models

import (
	"time"

	"gorm.io/gorm"
)

type FinancialGoal struct {
	ID            int            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint           `gorm:"not null" json:"user_id"`
	Name          string         `gorm:"not null" json:"name"`
	TargetAmount  int            `gorm:"not null" json:"target_amount"`
	CurrentAmount int            `gorm:"default:0" json:"current_amount"`
	Deadline      *time.Time     `json:"deadline"`
	Category      string         `gorm:"not null" json:"category"` // savings, travel, emergency, education, investment
	Icon          string         `json:"icon"`
	AutoAllocate  bool           `gorm:"default:false" json:"auto_allocate"` // Tự động phân bổ từ income
	AllocatePercent int          `gorm:"default:0" json:"allocate_percent"`  // % income tự động chuyển vào (0-100)
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
