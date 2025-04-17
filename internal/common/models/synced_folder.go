package models

import "time"

type SyncedFolder struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	FolderName  string    `gorm:"uniqueIndex;not null"`
	LastChecked time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	FullySynced bool      `gorm:"not null;default:false"`
	TotalFiles  int       `gorm:"not null;default:0"`
}
