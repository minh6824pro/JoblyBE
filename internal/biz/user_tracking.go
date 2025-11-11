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
)

type UserTracking struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"user_id" json:"user_id"`
	TrackingType TrackingType       `bson:"tracking_type" json:"tracking_type"`
	Metadata     interface{}        `bson:"metadata" json:"metadata"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

type UserTrackingRepo interface {
	CreateUserTracking(ctx context.Context, userTracking *UserTracking) (*UserTracking, error)
}

type UserTrackingUseCase struct {
	UserTrackingRepo UserTrackingRepo
	log              *log.Helper
}

func NewUserTrackingUseCase(userRepo UserTrackingRepo, logger log.Logger) *UserTrackingUseCase {
	return &UserTrackingUseCase{
		UserTrackingRepo: userRepo,
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
