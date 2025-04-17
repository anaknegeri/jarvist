package models

import (
	"time"
)

type ProcessedFile struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	Filename    string    `gorm:"uniqueIndex;not null"`
	DateFolder  string    `gorm:"not null"`
	ProcessedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	DataJSON    string
}
