package models

import (
	"time"

	"gorm.io/gorm"
)

type LogLevel string

const (
	LogLevelTrace   LogLevel = "TRACE"
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelError   LogLevel = "ERROR"
	LogLevelFatal   LogLevel = "FATAL"
)

type LogEntry struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Timestamp time.Time `gorm:"index;default:CURRENT_TIMESTAMP"`
	Level     string    `gorm:"index"`
	Component string    `gorm:"index"`
	Message   string
}

func (l *LogEntry) BeforeCreate(tx *gorm.DB) (err error) {
	if l.Timestamp.IsZero() {
		l.Timestamp = time.Now()
	}
	if l.Component == "" {
		l.Component = "system"
	}
	return
}
