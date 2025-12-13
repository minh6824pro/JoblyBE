package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrApplicationNotFound      = errors.New("application not found")
	ErrApplicationAlreadyExists = errors.New("user already applied for this job")
	ErrInvalidApplicationData   = errors.New("invalid application data")
	ErrUnauthorizedAction       = errors.New("unauthorized action")
)

// ApplicationStatus represents the status of a job application
type ApplicationStatus string

const (
	StatusApplied   ApplicationStatus = "applied"
	StatusReviewing ApplicationStatus = "reviewing"
	StatusAccepted  ApplicationStatus = "accepted"
	StatusRejected  ApplicationStatus = "rejected"
)

// JobApplication represents a job application entity
type JobApplication struct {
	ID            string
	UserID        string
	ResumeID      string
	JobID         string
	ApplicantName string
	University    string
	Status        ApplicationStatus
	AppliedAt     time.Time
	UpdatedAt     time.Time
	HRNote        string

	// Full detail (loaded on demand)
	ResumeDetail *ResumeDetail
	JobInfo      *JobApplicationJobInfo
}

// JobApplicationJobInfo contains basic job info for application
type JobApplicationJobInfo struct {
	ID          string
	Title       string
	CompanyName string
	Location    string
}

// JobApplicationRepo interface
type JobApplicationRepo interface {
	CreateApplication(ctx context.Context, app *JobApplication) (*JobApplication, error)
	GetApplication(ctx context.Context, id string) (*JobApplication, error)
	GetApplicationWithDetail(ctx context.Context, id string) (*JobApplication, error)
	ListUserApplications(ctx context.Context, userID string, page, pageSize int32) ([]*JobApplication, int32, error)
	ListJobApplications(ctx context.Context, jobID string, status ApplicationStatus, page, pageSize int32) ([]*JobApplication, int32, error)
	UpdateApplicationStatus(ctx context.Context, id string, status ApplicationStatus, hrNote string) error
	DeleteApplication(ctx context.Context, id string) error
	CheckExistingApplication(ctx context.Context, userID, jobID string) (bool, error)
}

// JobApplicationUseCase handles job application business logic
type JobApplicationUseCase struct {
	repo       JobApplicationRepo
	resumeRepo ResumeRepo
	jobRepo    JobPostingRepo
	log        *log.Helper
}

// NewJobApplicationUseCase creates a new job application use case
func NewJobApplicationUseCase(
	repo JobApplicationRepo,
	resumeRepo ResumeRepo,
	jobRepo JobPostingRepo,
	logger log.Logger,
) *JobApplicationUseCase {
	return &JobApplicationUseCase{
		repo:       repo,
		resumeRepo: resumeRepo,
		jobRepo:    jobRepo,
		log:        log.NewHelper(logger),
	}
}

// ApplyJob creates a new job application
func (uc *JobApplicationUseCase) ApplyJob(ctx context.Context, userID, jobID, resumeID string) (*JobApplication, error) {
	// Check if user already applied for this job
	exists, err := uc.repo.CheckExistingApplication(ctx, userID, jobID)
	if err != nil {
		uc.log.Errorf("failed to check existing application: %v", err)
		return nil, err
	}
	if exists {
		return nil, ErrApplicationAlreadyExists
	}

	// Verify resume exists and belongs to user
	resume, err := uc.resumeRepo.GetResume(ctx, resumeID)
	if err != nil {
		uc.log.Errorf("failed to get resume: %v", err)
		return nil, err
	}
	if resume.UserID != userID {
		return nil, ErrUnauthorizedAction
	}

	// Verify job exists
	_, err = uc.jobRepo.GetJobPosting(ctx, jobID)
	if err != nil {
		uc.log.Errorf("failed to get job posting: %v", err)
		return nil, err
	}

	// Extract basic info from resume
	applicantName := resume.ResumeDetail.Name
	university := ""
	if len(resume.ResumeDetail.Education) > 0 {
		university = resume.ResumeDetail.Education[0].Institution
	}

	// Create application
	now := time.Now()
	app := &JobApplication{
		UserID:        userID,
		ResumeID:      resumeID,
		JobID:         jobID,
		ApplicantName: applicantName,
		University:    university,
		Status:        StatusApplied,
		AppliedAt:     now,
		UpdatedAt:     now,
	}

	return uc.repo.CreateApplication(ctx, app)
}

// GetApplication retrieves basic application info
func (uc *JobApplicationUseCase) GetApplication(ctx context.Context, id string) (*JobApplication, error) {
	return uc.repo.GetApplication(ctx, id)
}

// GetApplicationWithDetail retrieves application with full resume detail
func (uc *JobApplicationUseCase) GetApplicationWithDetail(ctx context.Context, id string) (*JobApplication, error) {
	return uc.repo.GetApplicationWithDetail(ctx, id)
}

// ListUserApplications lists all applications for a user
func (uc *JobApplicationUseCase) ListUserApplications(ctx context.Context, userID string, page, pageSize int32) ([]*JobApplication, int32, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return uc.repo.ListUserApplications(ctx, userID, page, pageSize)
}

// ListJobApplications lists all applications for a job posting
func (uc *JobApplicationUseCase) ListJobApplications(ctx context.Context, jobID string, status ApplicationStatus, page, pageSize int32) ([]*JobApplication, int32, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return uc.repo.ListJobApplications(ctx, jobID, status, page, pageSize)
}

// UpdateApplicationStatus updates the status of an application (for HR)
func (uc *JobApplicationUseCase) UpdateApplicationStatus(ctx context.Context, id string, status ApplicationStatus, hrNote string) error {
	// Validate status
	validStatuses := map[ApplicationStatus]bool{
		StatusApplied:   true,
		StatusReviewing: true,
		StatusAccepted:  true,
		StatusRejected:  true,
	}
	if !validStatuses[status] {
		return ErrInvalidApplicationData
	}

	return uc.repo.UpdateApplicationStatus(ctx, id, status, hrNote)
}

// WithdrawApplication withdraws an application (for user)
func (uc *JobApplicationUseCase) WithdrawApplication(ctx context.Context, id, userID string) error {
	// Check if application exists and belongs to user
	app, err := uc.repo.GetApplication(ctx, id)
	if err != nil {
		return err
	}
	if app.UserID != userID {
		return ErrUnauthorizedAction
	}

	// Only allow withdrawal if status is "applied" or "reviewing"
	if app.Status != StatusApplied && app.Status != StatusReviewing {
		return errors.New("cannot withdraw application with current status")
	}

	return uc.repo.DeleteApplication(ctx, id)
}
