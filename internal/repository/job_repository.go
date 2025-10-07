package repository

import (
	"Jobly/api/handler/controller/httpresponse"
	entities "Jobly/internal/entity"
	"context"

	"gorm.io/gorm"
)

type JobGormRepository struct {
	db *gorm.DB
}

type JobRepository interface {
	GetJobList(ctx context.Context, page int, keywords []string) (httpresponse.ListJobResponse, error)
}

func NewJobGormRepository(db *gorm.DB) JobRepository {
	return &JobGormRepository{
		db: db,
	}
}

func (j JobGormRepository) GetJobList(ctx context.Context, page int, keywords []string) (httpresponse.ListJobResponse, error) {
	var jobs []entities.Job
	var jobResponses []httpresponse.JobResponse
	var totalItems int64

	// Pagination settings
	pageSize := 15
	offset := (page - 1) * pageSize

	// Build base query
	query := j.db.WithContext(ctx).Model(&entities.Job{})

	// Apply keyword search if keywords provided
	if len(keywords) > 0 {
		for _, keyword := range keywords {
			if keyword != "" {
				searchPattern := "%" + keyword + "%"
				query = query.Where(
					"title LIKE ? OR description LIKE ? OR requirements LIKE ? OR responsibilities LIKE ?",
					searchPattern, searchPattern, searchPattern, searchPattern,
				)
			}
		}
	}

	// Count total items with filters
	err := query.Count(&totalItems).Error

	if err != nil {
		return httpresponse.ListJobResponse{}, err
	}

	// Calculate total pages
	totalPages := int((totalItems + int64(pageSize) - 1) / int64(pageSize))

	// Query with preload Company, order by created_at desc, and pagination
	err = query.
		Preload("Company").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&jobs).Error

	if err != nil {
		return httpresponse.ListJobResponse{}, err
	}

	// Map entities to response
	for _, job := range jobs {
		jobResponse := httpresponse.JobResponse{
			ID:                    job.ID,
			Name:                  job.Company.Name,
			LogoURL:               job.Company.LogoURL,
			Website:               job.Company.Website,
			CompanyLocation:       job.Company.Location,
			Title:                 job.Title,
			Level:                 job.Level,
			JobType:               job.JobType,
			SalaryMin:             job.SalaryMin,
			SalaryMax:             job.SalaryMax,
			SalaryCurrency:        job.SalaryCurrency,
			JobLocation:           job.Location,
			PostedAt:              job.PostedAt,
			ExperienceRequirement: job.ExperienceRequirement,
			Description:           job.Description,
			Responsibilities:      job.Responsibilities,
			Requirements:          job.Requirements,
			Benefits:              job.Benefits,
			JobTech:               job.JobTech,
			CreatedAt:             job.CreatedAt,
		}
		jobResponses = append(jobResponses, jobResponse)
	}

	return httpresponse.ListJobResponse{
		Jobs:        jobResponses,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalItems:  totalItems,
	}, nil
}
