package data

import (
	"JobblyBE/internal/biz"
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
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
		UserID:       primitive.ObjectID{},
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
