package models

import (
	"time"

	"gorm.io/gorm"
)

type PendingMessage struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	Topic           string    `gorm:"type:text;index"`
	Payload         string    `gorm:"type:text"`
	Timestamp       time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	Sent            bool      `gorm:"default:false;index"`
	RetryCount      int       `gorm:"default:0"`
	ConnectionState bool      `gorm:"default:false"`
	ExtraInfo       string    `gorm:"type:text"`
}

func (pm *PendingMessage) BeforeCreate(tx *gorm.DB) (err error) {
	if pm.Timestamp.IsZero() {
		pm.Timestamp = time.Now()
	}
	return
}

func (pm *PendingMessage) Retry() {
	pm.RetryCount++
}

func (pm *PendingMessage) MarkAsSent() {
	pm.Sent = true
}
