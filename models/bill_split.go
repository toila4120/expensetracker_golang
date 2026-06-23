package models

import (
	"time"

	"gorm.io/gorm"
)

type BillSplit struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	SharedBillID  uint           `gorm:"not null;index" json:"shared_bill_id"`
	GroupMemberID uint           `gorm:"not null" json:"group_member_id"`
	Amount        int            `gorm:"not null" json:"amount"`
	IsSettled     bool           `gorm:"default:false" json:"is_settled"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
