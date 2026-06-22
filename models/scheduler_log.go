package models

import "time"

type SchedulerLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	LastRunAt time.Time `gorm:"not null"`
}
