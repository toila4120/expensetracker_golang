package models

import (
	"time"

	"gorm.io/datatypes"
)

type Notification struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Type      string         `gorm:"not null" json:"type"`
	Title     string         `gorm:"not null" json:"title"`
	Message   string         `gorm:"not null" json:"message"`
	IsRead    bool           `gorm:"default:false" json:"is_read"`
	Metadata  datatypes.JSON `json:"metadata"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
}
