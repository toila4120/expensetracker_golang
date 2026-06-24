package models

import (
	"time"

	"gorm.io/gorm"
)

type Group struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"not null" json:"name"`
	Description string         `json:"description"`
	Type        string         `gorm:"not null;default:'regular'" json:"type"` // regular | peer_to_peer
	CreatedBy   uint           `gorm:"not null" json:"created_by"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type GroupMember struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   uint           `gorm:"not null;index" json:"group_id"`
	UserID    *uint          `gorm:"index" json:"user_id"`
	GuestName string         `json:"guest_name"`
	Role      string         `gorm:"default:'member'" json:"role"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
