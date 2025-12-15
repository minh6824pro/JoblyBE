package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TrackingType string

const (
	TrackingJobFilter TrackingType = "tracking_job_filter"
	TrackingJDTOS     TrackingType = "tracking_user_jd_tos"
)

type UserTracking struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"user_id" json:"user_id"`
	TrackingType TrackingType       `bson:"tracking_type" json:"tracking_type"`
	Metadata     interface{}        `bson:"metadata" json:"metadata"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

type UserJDTOS struct {
	JobID       primitive.ObjectID `bson:"job_id" json:"job_id"`
	TimeOnSight int32              `bson:"time_on_sight" json:"time_on_sight"`
}
type UserTrackingRepo interface {
	CreateUserTracking(ctx context.Context, userTracking *UserTracking) (*UserTracking, error)
	FindAndUpdateUserTrackingJDTOS(ctx context.Context, userID, jobID primitive.ObjectID, additionalTime int32) (*UserTracking, error)
	GetMostViewedJobByUser(ctx context.Context, userID primitive.ObjectID) (*UserJDTOS, error)
	GetTopViewedJobsInLastWeek(ctx context.Context, userID primitive.ObjectID, limit int) ([]*UserJDTOS, error)
}

type UserTrackingUseCase struct {
	UserTrackingRepo UserTrackingRepo
	jobRepo          JobPostingRepo
	log              *log.Helper
}

func NewUserTrackingUseCase(userRepo UserTrackingRepo, jobRepo JobPostingRepo, logger log.Logger) *UserTrackingUseCase {
	return &UserTrackingUseCase{
		UserTrackingRepo: userRepo,
		jobRepo:          jobRepo,
		log:              log.NewHelper(logger),
	}
}

func (uc *UserTrackingUseCase) CreateUserTrackingJobFilter(ctx context.Context, userID string, filter *JobFilter) error {
	userIDObject, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	// Build metadata with only non-empty fields
	metadata := make(map[string]interface{})

	if filter.CompanyID != "" {
		metadata["company_id"] = filter.CompanyID
	}
	if filter.Location != "" {
		metadata["location"] = filter.Location
	}
	if filter.JobType != "" {
		metadata["job_type"] = string(filter.JobType)
	}
	if filter.Level != "" {
		metadata["level"] = string(filter.Level)
	}
	if filter.Keyword != "" {
		metadata["keyword"] = filter.Keyword
	}
	if len(filter.JobTech) > 0 {
		metadata["job_tech"] = filter.JobTech
	}

	// Only create tracking if there's at least one filter
	if len(metadata) == 0 {
		return nil // No filters to track
	}

	now := time.Now()
	userTracking := &UserTracking{
		UserID:       userIDObject,
		TrackingType: TrackingJobFilter,
		Metadata:     metadata,
		CreatedAt:    now,
	}
	_, err = uc.UserTrackingRepo.CreateUserTracking(ctx, userTracking)
	if err != nil {
		return err
	}
	return nil
}

func (uc *UserTrackingUseCase) CreateUserTrackingJDTOS(ctx context.Context, userID, JobID string, tOS int32) error {
	userIDObject, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}
	jobIDObject, err := primitive.ObjectIDFromHex(JobID)
	if err != nil {
		return err
	}

	// Check if job exists
	job, err := uc.jobRepo.GetJobPosting(ctx, JobID)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrJobNotFound
	}

	// Try to find and update existing tracking first
	_, err = uc.UserTrackingRepo.FindAndUpdateUserTrackingJDTOS(ctx, userIDObject, jobIDObject, tOS)
	if err == nil {
		// Successfully updated existing tracking
		return nil
	}

	// If not found, create new tracking
	now := time.Now()
	userTracking := &UserTracking{
		UserID:       userIDObject,
		TrackingType: TrackingJDTOS,
		Metadata: UserJDTOS{
			JobID:       jobIDObject,
			TimeOnSight: tOS,
		},
		CreatedAt: now,
	}
	_, err = uc.UserTrackingRepo.CreateUserTracking(ctx, userTracking)
	return err
}

// GetMostViewedJobByUser gets the job with highest time on sight for a user
func (uc *UserTrackingUseCase) GetMostViewedJobByUser(ctx context.Context, userID string) (*JobPosting, error) {
	userIDObject, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	// Get tracking with highest time on sight
	tracking, err := uc.UserTrackingRepo.GetMostViewedJobByUser(ctx, userIDObject)
	if err != nil {
		return nil, err
	}

	// Get the job details
	jobID := tracking.JobID.Hex()
	job, err := uc.jobRepo.GetJobPosting(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return job, nil
}
