package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const CollectionResumeParseJob = "resume_parse_job"

// ResumeParseStatus represents the status of a resume parsing job
type ResumeParseStatus string

const (
	ResumeParseStatusPending    ResumeParseStatus = "pending"
	ResumeParseStatusProcessing ResumeParseStatus = "processing"
	ResumeParseStatusCompleted  ResumeParseStatus = "completed"
	ResumeParseStatusFailed     ResumeParseStatus = "failed"
)

// ResumeParseJob represents a resume parsing job in the database
type ResumeParseJob struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	UserID       primitive.ObjectID `bson:"user_id"`
	FileName     string             `bson:"file_name"`
	FileData     []byte             `bson:"file_data"` // Store file temporarily
	Status       ResumeParseStatus  `bson:"status"`
	ErrorMessage string             `bson:"error_message,omitempty"`
	ResumeID     primitive.ObjectID `bson:"resume_id,omitempty"` // Set when completed
	CreatedAt    time.Time          `bson:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at"`
}

// ResumeParseJobRepo handles resume parse job database operations
type ResumeParseJobRepo struct {
	data *Data
	log  *log.Helper
}

// NewResumeParseJobRepo creates a new resume parse job repository
func NewResumeParseJobRepo(data *Data, logger log.Logger) *ResumeParseJobRepo {
	return &ResumeParseJobRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// Create creates a new resume parse job
func (r *ResumeParseJobRepo) Create(ctx context.Context, job *ResumeParseJob) (*ResumeParseJob, error) {
	job.ID = primitive.NewObjectID()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	_, err := r.data.db.Collection(CollectionResumeParseJob).InsertOne(ctx, job)
	if err != nil {
		r.log.Errorf("failed to create resume parse job: %v", err)
		return nil, err
	}

	return job, nil
}

// GetByID retrieves a resume parse job by ID
func (r *ResumeParseJobRepo) GetByID(ctx context.Context, id string) (*ResumeParseJob, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var job ResumeParseJob
	err = r.data.db.Collection(CollectionResumeParseJob).FindOne(ctx, bson.M{"_id": objID}).Decode(&job)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.log.Errorf("failed to get resume parse job: %v", err)
		return nil, err
	}

	return &job, nil
}

// UpdateStatus updates the status of a resume parse job
func (r *ResumeParseJobRepo) UpdateStatus(ctx context.Context, id string, status ResumeParseStatus, errorMsg string, resumeID string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if errorMsg != "" {
		update["$set"].(bson.M)["error_message"] = errorMsg
	}

	if resumeID != "" {
		resumeObjID, err := primitive.ObjectIDFromHex(resumeID)
		if err == nil {
			update["$set"].(bson.M)["resume_id"] = resumeObjID
		}
	}

	_, err = r.data.db.Collection(CollectionResumeParseJob).UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		r.log.Errorf("failed to update resume parse job status: %v", err)
		return err
	}

	return nil
}

// ClearFileData clears the file data after processing to save space
func (r *ResumeParseJobRepo) ClearFileData(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"file_data":  nil,
			"updated_at": time.Now(),
		},
	}

	_, err = r.data.db.Collection(CollectionResumeParseJob).UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		r.log.Errorf("failed to clear file data: %v", err)
		return err
	}

	return nil
}

// ListByUserID lists all resume parse jobs for a user
func (r *ResumeParseJobRepo) ListByUserID(ctx context.Context, userID string, page, pageSize int32) ([]*ResumeParseJob, int64, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, err
	}

	filter := bson.M{"user_id": userObjID}

	// Count total
	total, err := r.data.db.Collection(CollectionResumeParseJob).CountDocuments(ctx, filter)
	if err != nil {
		r.log.Errorf("failed to count resume parse jobs: %v", err)
		return nil, 0, err
	}

	// Find with pagination
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := r.data.db.Collection(CollectionResumeParseJob).Find(ctx, filter, opts)
	if err != nil {
		r.log.Errorf("failed to list resume parse jobs: %v", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var jobs []*ResumeParseJob
	if err := cursor.All(ctx, &jobs); err != nil {
		r.log.Errorf("failed to decode resume parse jobs: %v", err)
		return nil, 0, err
	}

	return jobs, total, nil
}
