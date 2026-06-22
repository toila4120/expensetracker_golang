// File: models/transaction.go
// Mục tiêu: Định nghĩa cấu trúc dữ liệu cho giao dịch (thu/chi).
//
// Nhiệm vụ của bạn:
// 1. Tạo struct `Transaction` gồm: ID, UserID (khóa ngoại), Type (thu hoặc chi), Amount (số tiền), Category (danh mục), Note (ghi chú), Date (ngày giao dịch).
// 2. Thiết lập quan hệ (Relationship) giữa User và Transaction (1 User có nhiều Transaction).
//
// Kiến thức cần học:
// - Has Many / Belongs To relationship trong GORM.
// - Kiểu dữ liệu thời gian (time.Time) trong Go.

package models

import (
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint   `gorm:"not null;ForeignKey:UserID" json:"user_id"`
	WalletID  *uint  `gorm:"index" json:"wallet_id"`
	Type      string `gorm:"not null" json:"type"`
	Amount    int    `gorm:"not null" json:"amount"`
	Category  string `gorm:"not null" json:"category"`
	Note      string `json:"description"`
	Date      time.Time      `gorm:"not null" json:"date"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
