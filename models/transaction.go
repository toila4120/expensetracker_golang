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
	ID        int    `gorm:"primaryKey;autoIncrement"`
	UserID    uint   `gorm:"not null;ForeignKey:UserID"`
	Type      string `gorm:"not null"`
	Amount    int    `gorm:"not null"`
	Category  string `gorm:"not null"`
	Note      string
	Date      time.Time      `gorm:"not null"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
