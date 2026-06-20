// File: models/user.go
// Mục tiêu: Định nghĩa cấu trúc dữ liệu cho người dùng.
//
// Nhiệm vụ của bạn:
// 1. Tạo một struct `User` bao gồm các trường: ID, Username, Email, Password, CreatedAt.
// 2. Định nghĩa các tag (tags) cho struct như `json:"email"` để định dạng lại JSON khi trả về client và `gorm:"unique"` để báo cho GORM biết ràng buộc trong DB (ví dụ email không được trùng).
//
// Kiến thức cần học:
// - Struct trong Golang.
// - Struct tags trong Golang (JSON tags, GORM tags).

package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Username  string    `gorm:"not null"`
	Email     string    `gorm:"unique;not null"`
	Password  string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
