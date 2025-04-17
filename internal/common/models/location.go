package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Location struct {
	ID          string  `json:"id" gorm:"type:text;primaryKey"`
	Name        string  `gorm:"not null" json:"name"`
	Description string  `gorm:"type:text" json:"description"`
	CreatedAt   string  `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt   *string `json:"deleted_at,omitempty"`
}

func (p *Location) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	p.CreatedAt = time.Now().Format(time.RFC3339)
	return
}

type LocationInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
