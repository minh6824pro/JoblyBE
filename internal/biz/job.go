package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrJobNotFound           = errors.New("job not found")
	ErrJobAlreadyExists      = errors.New("job already exists")
	ErrInvalidJobData        = errors.New("invalid job data")
	ErrJobExpired            = errors.New("job expired")
	ErrUnauthorizedJobAction = errors.New("unauthorized job action")
)

// Job types
type JobType string

const (
	FullTime   JobType = "FULL_TIME"
	PartTime   JobType = "PART_TIME"
	Contract   JobType = "CONTRACT"
	Internship JobType = "INTERNSHIP"
)

// Experience levels
type Level string

const (
	Entry  Level = "ENTRY"
	Junior Level = "JUNIOR"
	Mid    Level = "MID"
	Senior Level = "SENIOR"
	Lead   Level = "LEAD"
)

// JobPosting entity
type JobPosting struct {
	ID                    string
	CompanyID             string
	Company               *Company
	Title                 string
	Level                 Level
	JobType               JobType
	SalaryMin             float64
	SalaryMax             float64
	SalaryCurrency        string
	Location              string
	PostedAt              *time.Time
	ExperienceRequirement string
	Description           string
	Responsibilities      string
	Requirements          string
	Benefits              string
	JobTech               []string
	CreatedAt             time.Time
}

// JobPostingRepo interface
type JobPostingRepo interface {
	CreateJobPosting(ctx context.Context, job *JobPosting) (*JobPosting, error)
	UpdateJobPosting(ctx context.Context, job *JobPosting) error
	DeleteJobPosting(ctx context.Context, id string) error
	GetJobPosting(ctx context.Context, id string) (*JobPosting, error)
	ListJobPostings(ctx context.Context, filter *JobFilter, page, pageSize int32) ([]*JobPosting, int32, error)
}

// JobFilter for filtering and searching jobs
type JobFilter struct {
	CompanyID  string
	Location   string
	JobType    JobType
	Level      Level
	Keyword    string
	JobTech    []string
}

// JobPostingUseCase handles job posting business logic
type JobPostingUseCase struct {
	jobRepo     JobPostingRepo
	companyRepo CompanyRepo
	log         *log.Helper
}

// NewJobPostingUseCase creates a new job posting use case
func NewJobPostingUseCase(jobRepo JobPostingRepo, companyRepo CompanyRepo, logger log.Logger) *JobPostingUseCase {
	return &JobPostingUseCase{
		jobRepo:     jobRepo,
		companyRepo: companyRepo,
		log:         log.NewHelper(logger),
	}
}

// CreateJobPosting creates a new job posting
func (uc *JobPostingUseCase) CreateJobPosting(ctx context.Context, job *JobPosting) (*JobPosting, error) {
	uc.log.WithContext(ctx).Infof("CreateJobPosting: %s", job.Title)

	// Validate company exists
	company, err := uc.companyRepo.GetCompany(ctx, job.CompanyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, ErrCompanyNotFound
	}

	// Validate job data
	if err := uc.validateJobPosting(job); err != nil {
		return nil, err
	}

	// Create job posting
	createdJob, err := uc.jobRepo.CreateJobPosting(ctx, job)
	if err != nil {
		uc.log.Errorf("failed to create job posting: %v", err)
		return nil, err
	}

	// Attach company info
	createdJob.Company = company

	return createdJob, nil
}

// UpdateJobPosting updates an existing job posting
func (uc *JobPostingUseCase) UpdateJobPosting(ctx context.Context, job *JobPosting) (*JobPosting, error) {
	uc.log.WithContext(ctx).Infof("UpdateJobPosting: %s", job.ID)

	// Get existing job
	existingJob, err := uc.jobRepo.GetJobPosting(ctx, job.ID)
	if err != nil {
		return nil, err
	}
	if existingJob == nil {
		return nil, ErrJobNotFound
	}

	// Validate job data
	if err := uc.validateJobPosting(job); err != nil {
		return nil, err
	}

	// Update job posting
	if err := uc.jobRepo.UpdateJobPosting(ctx, job); err != nil {
		uc.log.Errorf("failed to update job posting: %v", err)
		return nil, err
	}

	// Get updated job with company info
	updatedJob, err := uc.jobRepo.GetJobPosting(ctx, job.ID)
	if err != nil {
		return nil, err
	}

	return updatedJob, nil
}

// DeleteJobPosting deletes a job posting
func (uc *JobPostingUseCase) DeleteJobPosting(ctx context.Context, id string) error {
	uc.log.WithContext(ctx).Infof("DeleteJobPosting: %s", id)

	// Get existing job
	existingJob, err := uc.jobRepo.GetJobPosting(ctx, id)
	if err != nil {
		return err
	}
	if existingJob == nil {
		return ErrJobNotFound
	}

	// Delete job posting
	if err := uc.jobRepo.DeleteJobPosting(ctx, id); err != nil {
		uc.log.Errorf("failed to delete job posting: %v", err)
		return err
	}

	return nil
}

// GetJobPosting retrieves a job posting by ID
func (uc *JobPostingUseCase) GetJobPosting(ctx context.Context, id string) (*JobPosting, error) {
	uc.log.WithContext(ctx).Infof("GetJobPosting: %s", id)

	job, err := uc.jobRepo.GetJobPosting(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrJobNotFound
	}

	return job, nil
}

// ListJobPostings lists job postings with filters and pagination
func (uc *JobPostingUseCase) ListJobPostings(ctx context.Context, filter *JobFilter, page, pageSize int32) ([]*JobPosting, int32, error) {
	uc.log.WithContext(ctx).Info("ListJobPostings")

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	jobs, total, err := uc.jobRepo.ListJobPostings(ctx, filter, page, pageSize)
	if err != nil {
		uc.log.Errorf("failed to list job postings: %v", err)
		return nil, 0, err
	}

	return jobs, total, nil
}

// validateJobPosting validates job posting data
func (uc *JobPostingUseCase) validateJobPosting(job *JobPosting) error {
	if job.Title == "" {
		return ErrInvalidJobData
	}
	if job.Description == "" {
		return ErrInvalidJobData
	}
	if job.CompanyID == "" {
		return ErrInvalidJobData
	}

	// Validate job type
	validJobTypes := map[JobType]bool{
		FullTime:   true,
		PartTime:   true,
		Contract:   true,
		Internship: true,
	}
	if !validJobTypes[job.JobType] {
		return ErrInvalidJobData
	}

	// Validate level
	validLevels := map[Level]bool{
		Entry:  true,
		Junior: true,
		Mid:    true,
		Senior: true,
		Lead:   true,
	}
	if !validLevels[job.Level] {
		return ErrInvalidJobData
	}

	return nil
}
