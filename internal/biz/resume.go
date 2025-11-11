package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// Resume represents a user's resume
type Resume struct {
	ID           string
	UserID       string
	ResumeDetail *ResumeDetail
	Version      int32
	CreatedAt    time.Time
}

type ResumeDetail struct {
	Name           string
	Email          string
	Phone          string
	Summary        string
	Skills         []string
	Education      *Education
	Experience     *Experience
	Certifications []string
	Languages      []string
}

type Education struct {
	Degree         string
	Institution    string
	GraduationYear string
}

type Experience struct {
	Title            string
	Company          string
	Duration         string
	Responsibilities []string
	Achievements     []string
}

// ResumeRepo is the interface for resume repository
type ResumeRepo interface {
	CreateResume(ctx context.Context, resume *Resume) (*Resume, error)
	UpdateResume(ctx context.Context, resume *Resume) (*Resume, error)
	GetResume(ctx context.Context, id string) (*Resume, error)
	ListResumes(ctx context.Context, userID string, page, pageSize int32) ([]*Resume, int32, error)
	DeleteResume(ctx context.Context, id string) error
}

// ResumeUseCase is the use case for resume operations
type ResumeUseCase struct {
	repo ResumeRepo
	log  *log.Helper
}

// NewResumeUseCase creates a new resume use case
func NewResumeUseCase(repo ResumeRepo, logger log.Logger) *ResumeUseCase {
	return &ResumeUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// CreateResume creates a new resume
func (uc *ResumeUseCase) CreateResume(ctx context.Context, resume *Resume) (*Resume, error) {
	// Validate resume
	if err := uc.validateResume(resume); err != nil {
		return nil, err
	}

	// Check if user already has a resume
	existingResumes, _, err := uc.repo.ListResumes(ctx, resume.UserID, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(existingResumes) > 0 {
		return nil, ErrResumeAlreadyExists
	}

	// Set timestamp and version
	resume.CreatedAt = time.Now()
	resume.Version = 1

	return uc.repo.CreateResume(ctx, resume)
}

// UpdateResume updates an existing resume
func (uc *ResumeUseCase) UpdateResume(ctx context.Context, resume *Resume) (*Resume, error) {
	// Validate resume
	if err := uc.validateResume(resume); err != nil {
		return nil, err
	}

	// Check if resume exists
	existing, err := uc.repo.GetResume(ctx, resume.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrResumeNotFound
	}

	// Check ownership
	if existing.UserID != resume.UserID {
		return nil, ErrUnauthorized
	}

	// Increment version and preserve creation time
	resume.Version = existing.Version + 1
	resume.CreatedAt = existing.CreatedAt

	return uc.repo.UpdateResume(ctx, resume)
}

// GetResume retrieves a resume by ID
func (uc *ResumeUseCase) GetResume(ctx context.Context, id, userID string) (*Resume, error) {
	resume, err := uc.repo.GetResume(ctx, id)
	if err != nil {
		return nil, err
	}
	if resume == nil {
		return nil, ErrResumeNotFound
	}

	// Check ownership
	if resume.UserID != userID {
		return nil, ErrUnauthorized
	}

	return resume, nil
}

// ListResumes lists all resumes for a user
func (uc *ResumeUseCase) ListResumes(ctx context.Context, userID string, page, pageSize int32) ([]*Resume, int32, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	return uc.repo.ListResumes(ctx, userID, page, pageSize)
}

// DeleteResume deletes a resume
func (uc *ResumeUseCase) DeleteResume(ctx context.Context, id, userID string) error {
	// Check if resume exists
	existing, err := uc.repo.GetResume(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrResumeNotFound
	}

	// Check ownership
	if existing.UserID != userID {
		return ErrUnauthorized
	}

	return uc.repo.DeleteResume(ctx, id)
}

// validateResume validates resume data
func (uc *ResumeUseCase) validateResume(resume *Resume) error {
	if resume.ResumeDetail == nil {
		return ErrInvalidResume
	}

	if resume.ResumeDetail.Name == "" {
		return ErrInvalidResume
	}

	if resume.ResumeDetail.Email == "" {
		return ErrInvalidResume
	}

	return nil
}

// Error definitions
var (
	ErrResumeNotFound      = errors.NotFound("RESUME_NOT_FOUND", "Resume not found")
	ErrInvalidResume       = errors.BadRequest("INVALID_RESUME", "Invalid resume data")
	ErrUnauthorized        = errors.Forbidden("UNAUTHORIZED", "You don't have permission to access this resume")
	ErrResumeAlreadyExists = errors.BadRequest("RESUME_ALREADY_EXISTS", "You already have a resume. Please update it instead of creating a new one")
)
