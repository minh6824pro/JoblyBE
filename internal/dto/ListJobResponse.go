package dto

import (
	"time"

	"gorm.io/datatypes"
)

type ListJobResponse struct {
	Jobs        []JobResponse `json:"jobs"`
	CurrentPage int           `json:"current_page"`
	TotalPages  int           `json:"total_pages"`
	TotalItems  int64         `json:"total_items"`
}

type JobResponse struct {
	ID                    uint           `json:"id"`
	Name                  string         `json:"name"`
	LogoURL               string         `json:"logo_url"`
	Website               string         `json:"website"`
	CompanyLocation       string         `json:"company_location"`
	Title                 string         `json:"title"`
	Level                 string         `json:"level"`
	JobType               string         `json:"job_type"`
	SalaryMin             float64        `json:"salary_min"`
	SalaryMax             float64        `json:"salary_max"`
	SalaryCurrency        string         `json:"salary_currency"`
	JobLocation           string         `json:"job_location"`
	PostedAt              *time.Time     `json:"posted_at"`
	ExperienceRequirement string         `json:"experience_requirement"`
	Description           string         `json:"description"`
	Responsibilities      string         `json:"responsibilities"`
	Requirements          string         `json:"requirements"`
	Benefits              string         `json:"benefits"`
	JobTech               datatypes.JSON `json:"job_tech"`
	CreatedAt             time.Time      `json:"created_at"`
}
