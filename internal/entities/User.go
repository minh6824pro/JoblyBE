package entities

import (
	"gorm.io/gorm"
	"time"
)

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

type User struct {
	UserID      uint   `gorm:"column:user_id;primaryKey;autoIncrement" json:"user_id"`
	FullName    string `gorm:"type:varchar(255)" json:"full_name"`
	Email       string `gorm:"type:varchar(255);unique;notnull" json:"email"`
	Password    string `gorm:"type:varchar(255);not null" json:"-"`
	PhoneNumber string `gorm:"type:varchar(255)" json:"phone_number"`
	Role        Role   `gorm:"type:enum('ADMIN','USER');default:'USER'" json:"role"`
	Active      uint8  `gorm:"type:TINYINT;default:1" json:"active"`

	// GORM default fields
	LastLogin *time.Time     `json:"-"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
