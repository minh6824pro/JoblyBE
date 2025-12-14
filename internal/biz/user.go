package biz

import (
	"context"
	"time"

	pb "JobblyBE/api/user/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrJobAlreadySaved = errors.New(400, pb.ErrorReason_JOB_ALREADY_SAVED.String(), "job already saved")
	ErrJobNotSaved     = errors.New(400, pb.ErrorReason_JOB_NOT_SAVED.String(), "job not saved")
	ErrJobNotFoundUser = errors.New(404, pb.ErrorReason_JOB_NOT_FOUND.String(), "job not found")
	ErrInvalidRequest  = errors.New(400, pb.ErrorReason_DATA_REQUEST_INVALID.String(), "invalid request data")
)

// User entity in business layer
type User struct {
	UserID      string
	FullName    string
	Email       string
	Password    string // hashed password
	PhoneNumber string
	Role        Role
	CompanyID   string // for HR role - linked company
	Active      bool
	LastLogin   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UserRepo interface định nghĩa các methods để tương tác với database
type UserRepo interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
	UpdateUser(ctx context.Context, user *User) error
	AddSavedJob(ctx context.Context, userID, jobID string) error
	RemoveSavedJob(ctx context.Context, userID, jobID string) error
	GetSavedJobIDs(ctx context.Context, userID string) ([]string, error)
}

// UserUseCase handles user business logic
type UserUseCase struct {
	userRepo UserRepo
	jobRepo  JobPostingRepo
	log      *log.Helper
}

// NewUserUseCase creates a new UserUseCase
func NewUserUseCase(userRepo UserRepo, jobRepo JobPostingRepo, logger log.Logger) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
		jobRepo:  jobRepo,
		log:      log.NewHelper(logger),
	}
}

// SaveJob saves a job for a user
func (uc *UserUseCase) SaveJob(ctx context.Context, userID, jobID string) error {
	uc.log.WithContext(ctx).Infof("SaveJob: user=%s, job=%s", userID, jobID)

	// Check if job exists
	job, err := uc.jobRepo.GetJobPosting(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrJobNotFoundUser
	}

	// Add to saved jobs
	err = uc.userRepo.AddSavedJob(ctx, userID, jobID)
	if err != nil {
		// Check if it's a "already saved" error
		if err.Error() == "job already saved" {
			return ErrJobAlreadySaved
		}
		return err
	}

	return nil
}

// UnsaveJob removes a job from user's saved jobs
func (uc *UserUseCase) UnsaveJob(ctx context.Context, userID, jobID string) error {
	uc.log.WithContext(ctx).Infof("UnsaveJob: user=%s, job=%s", userID, jobID)

	err := uc.userRepo.RemoveSavedJob(ctx, userID, jobID)
	if err != nil {
		return err
	}

	return nil
}

// GetSavedJobs gets all saved jobs for a user with pagination
func (uc *UserUseCase) GetSavedJobs(ctx context.Context, userID string, page, pageSize int32) ([]*JobPosting, int32, error) {
	uc.log.WithContext(ctx).Infof("GetSavedJobs: user=%s, page=%d, pageSize=%d", userID, page, pageSize)

	// Get saved job IDs
	jobIDs, err := uc.userRepo.GetSavedJobIDs(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	total := int32(len(jobIDs))
	if total == 0 {
		return []*JobPosting{}, 0, nil
	}

	// Apply pagination to job IDs
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return []*JobPosting{}, total, nil
	}
	if end > total {
		end = total
	}

	paginatedIDs := jobIDs[start:end]

	// Fetch jobs by IDs
	jobs := make([]*JobPosting, 0, len(paginatedIDs))
	for _, jobID := range paginatedIDs {
		job, err := uc.jobRepo.GetJobPosting(ctx, jobID)
		if err != nil {
			uc.log.Warnf("failed to get job %s: %v", jobID, err)
			continue
		}
		if job != nil {
			jobs = append(jobs, job)
		}
	}

	return jobs, total, nil
}
