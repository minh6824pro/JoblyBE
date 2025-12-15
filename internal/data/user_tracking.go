package data

import (
	"JobblyBE/internal/biz"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserTracking struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"user_id" json:"user_id"`
	TrackingType biz.TrackingType   `bson:"tracking_type" json:"tracking_type"`
	Metadata     interface{}        `bson:"metadata" json:"metadata"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

func (r *userTrackingRepo) toBiz(u *UserTracking) *biz.UserTracking {
	return &biz.UserTracking{
		ID:           u.ID,
		UserID:       u.UserID,
		TrackingType: u.TrackingType,
		Metadata:     u.Metadata,
		CreatedAt:    u.CreatedAt,
	}
}

type userTrackingRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserTrackingRepo(data *Data, logger log.Logger) biz.UserTrackingRepo {
	return &userTrackingRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *userTrackingRepo) CreateUserTracking(ctx context.Context, userTracking *biz.UserTracking) (*biz.UserTracking, error) {
	now := time.Now()
	ut := &UserTracking{
		UserID:       userTracking.UserID,
		TrackingType: userTracking.TrackingType,
		Metadata:     userTracking.Metadata,
		CreatedAt:    now,
	}
	result, err := r.data.db.Collection(CollectionUserTracking).InsertOne(ctx, ut)
	if err != nil {
		return nil, err
	}

	ut.ID = result.InsertedID.(primitive.ObjectID)
	return r.toBiz(ut), nil
}

func (r *userTrackingRepo) FindAndUpdateUserTrackingJDTOS(ctx context.Context, userID, jobID primitive.ObjectID, additionalTime int32) (*biz.UserTracking, error) {
	// Find existing tracking for this user and job
	filter := bson.M{
		"user_id":         userID,
		"tracking_type":   biz.TrackingJDTOS,
		"metadata.job_id": jobID,
	}

	// Increment time_on_sight
	update := bson.M{
		"$inc": bson.M{
			"metadata.time_on_sight": additionalTime,
		},
	}

	var ut UserTracking
	err := r.data.db.Collection(CollectionUserTracking).FindOneAndUpdate(ctx, filter, update).Decode(&ut)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		r.log.Errorf("failed to update user tracking: %v", err)
		return nil, err
	}

	return r.toBiz(&ut), nil
}

// GetMostViewedJobByUser gets the tracking with highest time_on_sight for a user
func (r *userTrackingRepo) GetMostViewedJobByUser(ctx context.Context, userID primitive.ObjectID) (*biz.UserJDTOS, error) {
	// Find all JDTOS trackings for this user
	filter := bson.M{
		"user_id":       userID,
		"tracking_type": biz.TrackingJDTOS,
	}
	// Sort by time_on_sight descending, limit 1
	opts := options.FindOne().SetSort(bson.M{"metadata.time_on_sight": -1})

	var ut UserTracking
	err := r.data.db.Collection(CollectionUserTracking).FindOne(ctx, filter, opts).Decode(&ut)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		r.log.Errorf("failed to get most viewed job: %v", err)
		return nil, err
	}

	// Extract metadata - handle both primitive.M and primitive.D
	var metadata primitive.M
	switch m := ut.Metadata.(type) {
	case primitive.M:
		metadata = m
	case primitive.D:
		metadata = m.Map()
	default:
		r.log.Errorf("unexpected metadata type: %T", ut.Metadata)
		return nil, biz.ErrInvalidRequest
	}

	jobID, ok := metadata["job_id"].(primitive.ObjectID)
	if !ok {
		r.log.Errorf("invalid job_id in metadata")
		return nil, biz.ErrInvalidRequest
	}

	timeOnSight, ok := metadata["time_on_sight"].(int32)
	if !ok {
		// Try int64 and convert
		if timeInt64, ok := metadata["time_on_sight"].(int64); ok {
			timeOnSight = int32(timeInt64)
		} else {
			r.log.Errorf("invalid time_on_sight in metadata")
			return nil, biz.ErrInvalidRequest
		}
	}

	return &biz.UserJDTOS{
		JobID:       jobID,
		TimeOnSight: timeOnSight,
	}, nil
}

// GetTopViewedJobsInLastWeek gets the top N trackings with highest time_on_sight in the last week
func (r *userTrackingRepo) GetTopViewedJobsInLastWeek(ctx context.Context, userID primitive.ObjectID, limit int) ([]*biz.UserJDTOS, error) {
	// Get date one week ago
	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	// Find all JDTOS trackings for this user in the last week
	filter := bson.M{
		"user_id":       userID,
		"tracking_type": biz.TrackingJDTOS,
		"created_at":    bson.M{"$gte": oneWeekAgo},
	}

	// Sort by time_on_sight descending
	opts := options.Find().
		SetSort(bson.M{"metadata.time_on_sight": -1}).
		SetLimit(int64(limit))

	cursor, err := r.data.db.Collection(CollectionUserTracking).Find(ctx, filter, opts)
	if err != nil {
		r.log.Errorf("failed to get top viewed jobs: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*biz.UserJDTOS
	for cursor.Next(ctx) {
		var ut UserTracking
		if err := cursor.Decode(&ut); err != nil {
			r.log.Errorf("failed to decode tracking: %v", err)
			continue
		}

		// Extract metadata
		var metadata primitive.M
		switch m := ut.Metadata.(type) {
		case primitive.M:
			metadata = m
		case primitive.D:
			metadata = m.Map()
		default:
			r.log.Errorf("unexpected metadata type: %T", ut.Metadata)
			continue
		}

		jobID, ok := metadata["job_id"].(primitive.ObjectID)
		if !ok {
			r.log.Errorf("invalid job_id in metadata")
			continue
		}

		timeOnSight, ok := metadata["time_on_sight"].(int32)
		if !ok {
			// Try int64 and convert
			if timeInt64, ok := metadata["time_on_sight"].(int64); ok {
				timeOnSight = int32(timeInt64)
			} else {
				r.log.Errorf("invalid time_on_sight in metadata")
				continue
			}
		}

		results = append(results, &biz.UserJDTOS{
			JobID:       jobID,
			TimeOnSight: timeOnSight,
		})
	}

	if err := cursor.Err(); err != nil {
		r.log.Errorf("cursor error: %v", err)
		return nil, err
	}

	return results, nil
}
