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

// JobApplication MongoDB model
type JobApplication struct {
	ID            primitive.ObjectID `bson:"_id"`
	UserID        primitive.ObjectID `bson:"user_id"`
	ResumeID      primitive.ObjectID `bson:"resume_id"`
	JobID         primitive.ObjectID `bson:"job_id"`
	ApplicantName string             `bson:"applicant_name"`
	University    string             `bson:"university"`
	Status        string             `bson:"status"`
	AppliedAt     time.Time          `bson:"applied_at"`
	UpdatedAt     time.Time          `bson:"updated_at"`
	HRNote        string             `bson:"hr_note"`
}

type jobApplicationRepo struct {
	data *Data
	log  *log.Helper
}

// NewJobApplicationRepo creates a new job application repository
func NewJobApplicationRepo(data *Data, logger log.Logger) biz.JobApplicationRepo {
	return &jobApplicationRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// CreateApplication creates a new job application
func (r *jobApplicationRepo) CreateApplication(ctx context.Context, app *biz.JobApplication) (*biz.JobApplication, error) {
	userObjID, err := primitive.ObjectIDFromHex(app.UserID)
	if err != nil {
		r.log.Errorf("invalid user ID: %v", err)
		return nil, err
	}

	resumeObjID, err := primitive.ObjectIDFromHex(app.ResumeID)
	if err != nil {
		r.log.Errorf("invalid resume ID: %v", err)
		return nil, err
	}

	jobObjID, err := primitive.ObjectIDFromHex(app.JobID)
	if err != nil {
		r.log.Errorf("invalid job ID: %v", err)
		return nil, err
	}

	appDoc := JobApplication{
		ID:            primitive.NewObjectID(),
		UserID:        userObjID,
		ResumeID:      resumeObjID,
		JobID:         jobObjID,
		ApplicantName: app.ApplicantName,
		University:    app.University,
		Status:        string(app.Status),
		AppliedAt:     app.AppliedAt,
		UpdatedAt:     app.UpdatedAt,
		HRNote:        app.HRNote,
	}

	collection := r.data.db.Collection(CollectionApplication)
	_, err = collection.InsertOne(ctx, appDoc)
	if err != nil {
		r.log.Errorf("failed to insert application: %v", err)
		return nil, err
	}

	app.ID = appDoc.ID.Hex()
	return app, nil
}

// GetApplication retrieves basic application info
func (r *jobApplicationRepo) GetApplication(ctx context.Context, id string) (*biz.JobApplication, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.log.Errorf("invalid application ID: %v", err)
		return nil, err
	}

	collection := r.data.db.Collection(CollectionApplication)
	var appDoc JobApplication
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&appDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, biz.ErrApplicationNotFound
		}
		r.log.Errorf("failed to get application: %v", err)
		return nil, err
	}

	return r.toApplicationBiz(&appDoc), nil
}

// GetApplicationWithDetail retrieves application with full resume detail and job info
func (r *jobApplicationRepo) GetApplicationWithDetail(ctx context.Context, id string) (*biz.JobApplication, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.log.Errorf("invalid application ID: %v", err)
		return nil, err
	}

	collection := r.data.db.Collection(CollectionApplication)

	// Use aggregation to join with resumes and jobs
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": objID}}},
		// Lookup resume detail from users collection
		{{Key: "$lookup", Value: bson.M{
			"from":         CollectionUser,
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "user",
		}}},
		{{Key: "$unwind", Value: "$user"}},
		// Lookup job info
		{{Key: "$lookup", Value: bson.M{
			"from":         CollectionJobPosting,
			"localField":   "job_id",
			"foreignField": "_id",
			"as":           "job",
		}}},
		{{Key: "$unwind", Value: "$job"}},
		// Lookup company info for job
		{{Key: "$lookup", Value: bson.M{
			"from":         CollectionCompany,
			"localField":   "job.company_id",
			"foreignField": "_id",
			"as":           "company",
		}}},
		{{Key: "$unwind", Value: "$company"}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		r.log.Errorf("failed to aggregate application: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return nil, biz.ErrApplicationNotFound
	}

	var result struct {
		JobApplication `bson:",inline"`
		User           struct {
			Resume []Resume `bson:"resume"`
		} `bson:"user"`
		Job struct {
			ID       primitive.ObjectID `bson:"_id"`
			Title    string             `bson:"title"`
			Location string             `bson:"location"`
		} `bson:"job"`
		Company struct {
			Name string `bson:"name"`
		} `bson:"company"`
	}

	if err := cursor.Decode(&result); err != nil {
		r.log.Errorf("failed to decode application: %v", err)
		return nil, err
	}

	app := r.toApplicationBiz(&result.JobApplication)

	// Find the matching resume in user's resume array
	for _, resume := range result.User.Resume {
		if resume.ID == result.JobApplication.ResumeID {
			app.ResumeDetail = r.toResumeDetailBiz(&resume.ResumeDetail)
			break
		}
	}

	// Add job info
	app.JobInfo = &biz.JobApplicationJobInfo{
		ID:          result.Job.ID.Hex(),
		Title:       result.Job.Title,
		CompanyName: result.Company.Name,
		Location:    result.Job.Location,
	}

	return app, nil
}

// ListUserApplications lists all applications for a user
func (r *jobApplicationRepo) ListUserApplications(ctx context.Context, userID string, page, pageSize int32) ([]*biz.JobApplication, int32, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		r.log.Errorf("invalid user ID: %v", err)
		return nil, 0, err
	}

	collection := r.data.db.Collection(CollectionApplication)
	filter := bson.M{"user_id": userObjID}

	// Get total count
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		r.log.Errorf("failed to count applications: %v", err)
		return nil, 0, err
	}

	// Use aggregation to join with jobs and companies
	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$sort", Value: bson.M{"applied_at": -1}}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: limit}},
		// Lookup job info
		{{Key: "$lookup", Value: bson.M{
			"from":         CollectionJobPosting,
			"localField":   "job_id",
			"foreignField": "_id",
			"as":           "job",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$job",
			"preserveNullAndEmptyArrays": true,
		}}},
		// Lookup company info for job
		{{Key: "$lookup", Value: bson.M{
			"from":         CollectionCompany,
			"localField":   "job.company_id",
			"foreignField": "_id",
			"as":           "company",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$company",
			"preserveNullAndEmptyArrays": true,
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		r.log.Errorf("failed to aggregate applications: %v", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var applications []*biz.JobApplication
	for cursor.Next(ctx) {
		var result struct {
			JobApplication `bson:",inline"`
			Job            struct {
				ID       primitive.ObjectID `bson:"_id"`
				Title    string             `bson:"title"`
				Location string             `bson:"location"`
			} `bson:"job"`
			Company struct {
				Name string `bson:"name"`
			} `bson:"company"`
		}

		if err := cursor.Decode(&result); err != nil {
			r.log.Errorf("failed to decode application: %v", err)
			continue
		}

		app := r.toApplicationBiz(&result.JobApplication)

		// Add job info if available
		if result.Job.ID != primitive.NilObjectID {
			app.JobInfo = &biz.JobApplicationJobInfo{
				ID:          result.Job.ID.Hex(),
				Title:       result.Job.Title,
				CompanyName: result.Company.Name,
				Location:    result.Job.Location,
			}
		}

		applications = append(applications, app)
	}

	return applications, int32(total), nil
}

// ListJobApplications lists all applications for a job posting
func (r *jobApplicationRepo) ListJobApplications(ctx context.Context, jobID string, status biz.ApplicationStatus, page, pageSize int32) ([]*biz.JobApplication, int32, error) {
	jobObjID, err := primitive.ObjectIDFromHex(jobID)
	if err != nil {
		r.log.Errorf("invalid job ID: %v", err)
		return nil, 0, err
	}

	collection := r.data.db.Collection(CollectionApplication)
	filter := bson.M{"job_id": jobObjID}

	// Add status filter if provided
	if status != "" {
		filter["status"] = string(status)
	}

	// Get total count
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		r.log.Errorf("failed to count applications: %v", err)
		return nil, 0, err
	}

	// Find with pagination
	opts := options.Find().
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize)).
		SetSort(bson.M{"applied_at": -1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		r.log.Errorf("failed to find applications: %v", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var applications []*biz.JobApplication
	for cursor.Next(ctx) {
		var appDoc JobApplication
		if err := cursor.Decode(&appDoc); err != nil {
			r.log.Errorf("failed to decode application: %v", err)
			continue
		}
		applications = append(applications, r.toApplicationBiz(&appDoc))
	}

	return applications, int32(total), nil
}

// UpdateApplicationStatus updates the status of an application
func (r *jobApplicationRepo) UpdateApplicationStatus(ctx context.Context, id string, status biz.ApplicationStatus, hrNote string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.log.Errorf("invalid application ID: %v", err)
		return err
	}

	collection := r.data.db.Collection(CollectionApplication)
	update := bson.M{
		"$set": bson.M{
			"status":     string(status),
			"hr_note":    hrNote,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		r.log.Errorf("failed to update application: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		return biz.ErrApplicationNotFound
	}

	return nil
}

// DeleteApplication deletes an application
func (r *jobApplicationRepo) DeleteApplication(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.log.Errorf("invalid application ID: %v", err)
		return err
	}

	collection := r.data.db.Collection(CollectionApplication)
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		r.log.Errorf("failed to delete application: %v", err)
		return err
	}

	if result.DeletedCount == 0 {
		return biz.ErrApplicationNotFound
	}

	return nil
}

// CheckExistingApplication checks if user already applied for a job
func (r *jobApplicationRepo) CheckExistingApplication(ctx context.Context, userID, jobID string) (bool, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return false, err
	}

	jobObjID, err := primitive.ObjectIDFromHex(jobID)
	if err != nil {
		return false, err
	}

	collection := r.data.db.Collection(CollectionApplication)
	count, err := collection.CountDocuments(ctx, bson.M{
		"user_id": userObjID,
		"job_id":  jobObjID,
	})
	if err != nil {
		r.log.Errorf("failed to check existing application: %v", err)
		return false, err
	}

	return count > 0, nil
}

// Helper functions

func (r *jobApplicationRepo) toApplicationBiz(doc *JobApplication) *biz.JobApplication {
	return &biz.JobApplication{
		ID:            doc.ID.Hex(),
		UserID:        doc.UserID.Hex(),
		ResumeID:      doc.ResumeID.Hex(),
		JobID:         doc.JobID.Hex(),
		ApplicantName: doc.ApplicantName,
		University:    doc.University,
		Status:        biz.ApplicationStatus(doc.Status),
		AppliedAt:     doc.AppliedAt,
		UpdatedAt:     doc.UpdatedAt,
		HRNote:        doc.HRNote,
	}
}

func (r *jobApplicationRepo) toResumeDetailBiz(doc *ResumeDetail) *biz.ResumeDetail {
	educations := make([]*biz.Education, len(doc.Education))
	for i, edu := range doc.Education {
		educations[i] = &biz.Education{
			Degree:         edu.Degree,
			Institution:    edu.Institution,
			GraduationYear: edu.GraduationYear,
		}
	}

	experiences := make([]*biz.Experience, len(doc.Experience))
	for i, exp := range doc.Experience {
		experiences[i] = &biz.Experience{
			Title:            exp.Title,
			Company:          exp.Company,
			Duration:         exp.Duration,
			Responsibilities: exp.Responsibilities,
			Achievements:     exp.Achievements,
		}
	}

	return &biz.ResumeDetail{
		Name:           doc.Name,
		Email:          doc.Email,
		Phone:          doc.Phone,
		Summary:        doc.Summary,
		Skills:         doc.Skills,
		Education:      educations,
		Experience:     experiences,
		Certifications: doc.Certifications,
		Languages:      doc.Languages,
	}
}
