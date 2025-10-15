package entities

import (
	"gorm.io/datatypes"

	"time"
)

type Job struct {
	ID                    uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	CompanyID             uint       `json:"company_id"`
	Title                 string     `gorm:"size:255;not null" json:"title"`
	Level                 string     `gorm:"size:50" json:"level"`
	JobType               string     `gorm:"size:100" json:"job_type"`
	SalaryMin             float64    `json:"salary_min"`
	SalaryMax             float64    `json:"salary_max"`
	SalaryCurrency        string     `gorm:"size:10" json:"salary_currency"`
	Location              string     `gorm:"size:255" json:"location"`
	PostedAt              *time.Time `json:"posted_at"`
	ExperienceRequirement string     `gorm:"size:255" json:"experience_requirement"`
	Description           string     `gorm:"type:text" json:"description"`
	Responsibilities      string     `gorm:"type:text" json:"responsibilities"`
	Requirements          string     `gorm:"type:text" json:"requirements"`
	Benefits              string     `gorm:"type:text" json:"benefits"`

	JobTech datatypes.JSON `gorm:"type:json" json:"job_tech"`

	Company   Company   `gorm:"foreignKey:CompanyID;references:ID" json:"company"`
	CreatedAt time.Time `json:"created_at"`
}
