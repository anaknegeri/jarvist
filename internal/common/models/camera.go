package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Camera struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"`
	UUID        string  `json:"uuid" gorm:"type:text"`
	Name        string  `gorm:"not null"`
	LocationID  string  `gorm:"type:text;not null;index"`
	Description string  `gorm:"type:text"`
	Tags        string  `gorm:"type:text"`
	Schema      string  `gorm:"not null"`
	Host        string  `gorm:"not null"`
	Port        int     `gorm:"default:0"`
	Path        string  `gorm:"type:text"`
	Username    string  `gorm:"type:text"`
	Password    string  `gorm:"type:text"`
	ImageData   string  `gorm:"type:text"`
	Direction   string  `gorm:"type:text"`
	Status      string  `gorm:"type:text"`
	Payload     string  `gorm:"type:json"`
	CreatedAt   string  `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt   *string `json:"deleted_at,omitempty"`

	IsConnected   *bool   `gorm:"-" json:"is_connected,omitempty"`
	LastChecked   *string `gorm:"-" json:"last_checked,omitempty"`
	StatusMessage *string `gorm:"-" json:"status_message,omitempty"`

	Location Location `gorm:"foreignKey:LocationID;constraint:OnDelete:SET NULL"`
}

func (p *Camera) BeforeCreate(tx *gorm.DB) (err error) {
	if p.UUID == "" {
		p.UUID = uuid.New().String()
	}
	p.CreatedAt = time.Now().Format(time.RFC3339)
	return
}

type CoordLocation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type LineData struct {
	Start     CoordLocation `json:"start"`
	End       CoordLocation `json:"end"`
	Direction string        `json:"direction"`
	Color     string        `json:"color"`
}

type CameraInput struct {
	Name        string     `json:"name"`
	Location    string     `json:"location"`
	Schema      string     `json:"schema"`
	Host        string     `json:"host"`
	Port        int        `json:"port"`
	Path        string     `json:"path"`
	Username    string     `json:"username"`
	Password    string     `json:"password"`
	ImageData   string     `json:"image_data"`
	Direction   string     `json:"direction"`
	Description string     `json:"description"`
	Tags        string     `json:"tags"`
	Lines       []LineData `json:"lines"`
}

type CameraResponse struct {
	Camera Camera     `json:"camera"`
	Lines  []LineData `json:"lines,omitempty"`
}

type LineJson struct {
	START [2]float64 `json:"START"`
	END   [2]float64 `json:"END"`
}

type Config struct {
	IP              string     `json:"IP"`
	ID              uint       `json:"ID"`
	UUID            string     `json:"UUID"`
	PORT            int        `json:"PORT"`
	IS_INSIDE_Y     bool       `json:"IS_INSIDE_Y"`
	IS_INSIDE_UNDER bool       `json:"IS_INSIDE_UNDER"`
	IS_INSIDE_LEFT  bool       `json:"IS_INSIDE_LEFT"`
	LINE            LineJson   `json:"LINE"`
	LINES           []LineJson `json:"LINES"`
}

type CameraConfig struct {
	TENANT_ID    string   `json:"TENANT_ID"`
	SITE_ID      int      `json:"SITE_ID"`
	CCTV_NUMBER  int      `json:"CCTV_NUMBER"`
	FRAME_HEIGHT int      `json:"FRAME_HEIGHT"`
	FRAME_WIDTH  int      `json:"FRAME_WIDTH"`
	CONFIG       []Config `json:"CONFIG"`
}
