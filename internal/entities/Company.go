package entities

import "time"

type Company struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name     string `gorm:"size:255;not null" json:"name"`
	LogoURL  string `gorm:"size:500" json:"logo_url"`
	Website  string `gorm:"size:500" json:"website"`
	Location string `gorm:"size:255" json:"location"`

	CreatedAt time.Time `json:"created_at"`
}
