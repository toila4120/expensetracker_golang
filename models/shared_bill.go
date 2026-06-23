package models

import (
	"time"

	"gorm.io/gorm"
)

type SharedBill struct {
	ID              uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID         uint           `gorm:"not null;index" json:"group_id"`
	PayerID         uint           `gorm:"not null" json:"payer_id"`
	CreatorID       uint           `gorm:"not null" json:"creator_id"`
	Amount          int            `gorm:"not null" json:"amount"`
	Category        string         `gorm:"not null" json:"category"`
	Description     string         `json:"description"`
	SplitMethod     string         `gorm:"not null" json:"split_method"`
	TransactionDate time.Time      `gorm:"not null" json:"transaction_date"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}
