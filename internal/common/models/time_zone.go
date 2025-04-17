package models

type TimeZone struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Zone      string `gorm:"unique;not null"`
	UTCOffset string `gorm:"not null"`
	Name      string `gorm:"not null"`
}

type TimeZoneResponse struct {
	Success   bool       `json:"success"`
	Error     string     `json:"error,omitempty"`
	TimeZone  *TimeZone  `json:"timezone,omitempty"`
	TimeZones []TimeZone `json:"timezones,omitempty"`
}
