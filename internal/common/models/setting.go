package models

type Setting struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Key   string `gorm:"uniqueIndex;not null"`
	Value string `gorm:"not null"`
}

type SettingInput struct {
	SiteCode        string `json:"site_code"`
	SiteName        string `json:"site_name"`
	SiteCategory    int    `json:"site_category"`
	DefaultTimezone string `json:"default_timezone"`
}
