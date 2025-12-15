package data

import (
	"context"
	"time"

	"JobblyBE/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NotificationModel represents the MongoDB notification document
type NotificationModel struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"user_id"`
	Title     string             `bson:"title"`
	Content   string             `bson:"content"`
	Type      string             `bson:"type"`
	ObjectID  string             `bson:"object_id"` // Reference to related object
	IsRead    bool               `bson:"is_read"`
	CreatedAt time.Time          `bson:"created_at"`
}

type notificationRepo struct {
	data *Data
	log  *log.Helper
	coll *mongo.Collection
}

// NewNotificationRepo creates a new notification repository
func NewNotificationRepo(data *Data, logger log.Logger) biz.NotificationRepo {
	return &notificationRepo{
		data: data,
		log:  log.NewHelper(logger),
		coll: data.db.Collection("notifications"),
	}
}

func (r *notificationRepo) CreateNotification(ctx context.Context, notification *biz.Notification) (*biz.Notification, error) {
	userID, err := primitive.ObjectIDFromHex(notification.UserID)
	if err != nil {
		return nil, err
	}

	model := &NotificationModel{
		UserID:    userID,
		Title:     notification.Title,
		Content:   notification.Content,
		Type:      notification.Type,
		ObjectID:  notification.ObjectID,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	result, err := r.coll.InsertOne(ctx, model)
	if err != nil {
		return nil, err
	}

	notification.ID = result.InsertedID.(primitive.ObjectID).Hex()
	notification.CreatedAt = model.CreatedAt
	return notification, nil
}

func (r *notificationRepo) GetNotificationsByUserID(ctx context.Context, userID string, page, pageSize int32) ([]*biz.Notification, int32, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, err
	}

	filter := bson.M{"user_id": userObjID}

	// Get total count
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Pagination
	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var notifications []*biz.Notification
	for cursor.Next(ctx) {
		var model NotificationModel
		if err := cursor.Decode(&model); err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, convertNotificationToBiz(&model))
	}

	return notifications, int32(total), nil
}

func (r *notificationRepo) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	notifObjID, err := primitive.ObjectIDFromHex(notificationID)
	if err != nil {
		return err
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":     notifObjID,
		"user_id": userObjID,
	}
	update := bson.M{"$set": bson.M{"is_read": true}}

	_, err = r.coll.UpdateOne(ctx, filter, update)
	return err
}

func (r *notificationRepo) MarkAllAsRead(ctx context.Context, userID string) error {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	filter := bson.M{
		"user_id": userObjID,
		"is_read": false,
	}
	update := bson.M{"$set": bson.M{"is_read": true}}

	_, err = r.coll.UpdateMany(ctx, filter, update)
	return err
}

func (r *notificationRepo) GetUnreadCount(ctx context.Context, userID string) (int32, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return 0, err
	}

	filter := bson.M{
		"user_id": userObjID,
		"is_read": false,
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int32(count), nil
}

func convertNotificationToBiz(model *NotificationModel) *biz.Notification {
	return &biz.Notification{
		ID:        model.ID.Hex(),
		UserID:    model.UserID.Hex(),
		Title:     model.Title,
		Content:   model.Content,
		Type:      model.Type,
		ObjectID:  model.ObjectID,
		IsRead:    model.IsRead,
		CreatedAt: model.CreatedAt,
	}
}
