package models

import (
	"time"

	"gorm.io/gorm"
)

type Settlement struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   uint           `gorm:"not null;index" json:"group_id"`
	FromID    uint           `gorm:"not null" json:"from_id"`
	ToID      uint           `gorm:"not null" json:"to_id"`
	Amount    int            `gorm:"not null" json:"amount"`
	Note      string         `json:"note"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
