package data

import (
	"JobblyBE/internal/biz"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// JobPosting struct for MongoDB
type JobPosting struct {
	ID                    primitive.ObjectID `bson:"_id,omitempty"`
	CompanyID             primitive.ObjectID `bson:"company_id"`
	Title                 string             `bson:"title"`
	Level                 string             `bson:"level"`
	JobType               string             `bson:"job_type"`
	SalaryMin             float64            `bson:"salary_min"`
	SalaryMax             float64            `bson:"salary_max"`
	SalaryCurrency        string             `bson:"salary_currency"`
	Location              string             `bson:"location"`
	PostedAt              *time.Time         `bson:"posted_at,omitempty"`
	ExperienceRequirement string             `bson:"experience_requirement"`
	Description           string             `bson:"description"`
	Responsibilities      string             `bson:"responsibilities"`
	Requirements          string             `bson:"requirements"`
	Benefits              string             `bson:"benefits"`
	JobTech               []string           `bson:"job_tech"`
	CreatedAt             time.Time          `bson:"created_at"`
}

type jobPostingRepo struct {
	data *Data
	log  *log.Helper
}

// NewJobPostingRepo creates a new job posting repository
func NewJobPostingRepo(data *Data, logger log.Logger) biz.JobPostingRepo {
	return &jobPostingRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// CreateJobPosting creates a new job posting
func (r *jobPostingRepo) CreateJobPosting(ctx context.Context, job *biz.JobPosting) (*biz.JobPosting, error) {
	now := time.Now()

	// Convert company ID to ObjectID
	companyObjID, err := primitive.ObjectIDFromHex(job.CompanyID)
	if err != nil {
		return nil, err
	}

	dbJob := &JobPosting{
		CompanyID:             companyObjID,
		Title:                 job.Title,
		Level:                 string(job.Level),
		JobType:               string(job.JobType),
		SalaryMin:             job.SalaryMin,
		SalaryMax:             job.SalaryMax,
		SalaryCurrency:        job.SalaryCurrency,
		Location:              job.Location,
		PostedAt:              job.PostedAt,
		ExperienceRequirement: job.ExperienceRequirement,
		Description:           job.Description,
		Responsibilities:      job.Responsibilities,
		Requirements:          job.Requirements,
		Benefits:              job.Benefits,
		JobTech:               job.JobTech,
		CreatedAt:             now,
	}

	result, err := r.data.db.Collection(CollectionJobPosting).InsertOne(ctx, dbJob)
	if err != nil {
		r.log.Errorf("failed to create job posting: %v", err)
		return nil, err
	}

	dbJob.ID = result.InsertedID.(primitive.ObjectID)
	return r.toBiz(dbJob), nil
}

// UpdateJobPosting updates an existing job posting
func (r *jobPostingRepo) UpdateJobPosting(ctx context.Context, job *biz.JobPosting) error {
	objID, err := primitive.ObjectIDFromHex(job.ID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"title":                  job.Title,
			"level":                  string(job.Level),
			"job_type":               string(job.JobType),
			"salary_min":             job.SalaryMin,
			"salary_max":             job.SalaryMax,
			"salary_currency":        job.SalaryCurrency,
			"location":               job.Location,
			"posted_at":              job.PostedAt,
			"experience_requirement": job.ExperienceRequirement,
			"description":            job.Description,
			"responsibilities":       job.Responsibilities,
			"requirements":           job.Requirements,
			"benefits":               job.Benefits,
			"job_tech":               job.JobTech,
		},
	}

	_, err = r.data.db.Collection(CollectionJobPosting).UpdateOne(
		ctx,
		bson.M{"_id": objID},
		update,
	)
	if err != nil {
		r.log.Errorf("failed to update job posting: %v", err)
		return err
	}

	return nil
}

// DeleteJobPosting deletes a job posting
func (r *jobPostingRepo) DeleteJobPosting(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.data.db.Collection(CollectionJobPosting).DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		r.log.Errorf("failed to delete job posting: %v", err)
		return err
	}

	return nil
}

// GetJobPosting retrieves a job posting by ID with company info
func (r *jobPostingRepo) GetJobPosting(ctx context.Context, id string) (*biz.JobPosting, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	// Use aggregation to join with company collection
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": objID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "companies",
			"localField":   "company_id",
			"foreignField": "_id",
			"as":           "company",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$company",
			"preserveNullAndEmptyArrays": true,
		}}},
	}

	cursor, err := r.data.db.Collection(CollectionJobPosting).Aggregate(ctx, pipeline)
	if err != nil {
		r.log.Errorf("failed to get job posting: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	type JobWithCompany struct {
		JobPosting `bson:",inline"`
		Company    *Company `bson:"company"`
	}

	var result JobWithCompany
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}

		bizJob := r.toBiz(&result.JobPosting)
		if result.Company != nil {
			companyRepo := &companyRepo{data: r.data, log: r.log}
			bizJob.Company = companyRepo.toBiz(result.Company)
		}
		return bizJob, nil
	}

	return nil, nil // Not found
}

// ListJobPostings lists job postings with filters and pagination
func (r *jobPostingRepo) ListJobPostings(ctx context.Context, filter *biz.JobFilter, page, pageSize int32) ([]*biz.JobPosting, int32, error) {
	// Build filter query
	query := bson.M{}

	if filter != nil {
		if filter.CompanyID != "" {
			companyObjID, err := primitive.ObjectIDFromHex(filter.CompanyID)
			if err == nil {
				query["company_id"] = companyObjID
			}
		}
		if filter.Location != "" {
			query["location"] = bson.M{"$regex": filter.Location, "$options": "i"}
		}
		if filter.JobType != "" {
			// Case-insensitive match for job_type
			query["job_type"] = bson.M{"$regex": "^" + string(filter.JobType) + "$", "$options": "i"}
		}
		if filter.Level != "" {
			// Case-insensitive match for level
			query["level"] = bson.M{"$regex": "^" + string(filter.Level) + "$", "$options": "i"}
		}
		if filter.Keyword != "" {
			query["$or"] = []bson.M{
				{"title": bson.M{"$regex": filter.Keyword, "$options": "i"}},
				{"description": bson.M{"$regex": filter.Keyword, "$options": "i"}},
			}
		}
		if len(filter.JobTech) > 0 {
			// Case-insensitive match for job_tech array
			techRegexes := make([]bson.M, len(filter.JobTech))
			for i, tech := range filter.JobTech {
				techRegexes[i] = bson.M{"$regex": "^" + tech + "$", "$options": "i"}
			}
			query["job_tech"] = bson.M{"$in": techRegexes}
		}
	}

	// Count total
	total, err := r.data.db.Collection(CollectionJobPosting).CountDocuments(ctx, query)
	if err != nil {
		r.log.Errorf("failed to count job postings: %v", err)
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * pageSize

	// Use aggregation to join with company
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		{{Key: "$sort", Value: bson.M{"created_at": -1}}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: pageSize}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "companies",
			"localField":   "company_id",
			"foreignField": "_id",
			"as":           "company",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$company",
			"preserveNullAndEmptyArrays": true,
		}}},
	}

	cursor, err := r.data.db.Collection(CollectionJobPosting).Aggregate(ctx, pipeline)
	if err != nil {
		r.log.Errorf("failed to list job postings: %v", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	type JobWithCompany struct {
		JobPosting `bson:",inline"`
		Company    *Company `bson:"company"`
	}

	var jobs []*biz.JobPosting
	for cursor.Next(ctx) {
		var result JobWithCompany
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		bizJob := r.toBiz(&result.JobPosting)
		if result.Company != nil {
			companyRepo := &companyRepo{data: r.data, log: r.log}
			bizJob.Company = companyRepo.toBiz(result.Company)
		}
		jobs = append(jobs, bizJob)
	}

	return jobs, int32(total), nil
}

// toBiz converts data layer JobPosting to biz layer JobPosting
func (r *jobPostingRepo) toBiz(j *JobPosting) *biz.JobPosting {
	return &biz.JobPosting{
		ID:                    j.ID.Hex(),
		CompanyID:             j.CompanyID.Hex(),
		Title:                 j.Title,
		Level:                 biz.Level(j.Level),
		JobType:               biz.JobType(j.JobType),
		SalaryMin:             j.SalaryMin,
		SalaryMax:             j.SalaryMax,
		SalaryCurrency:        j.SalaryCurrency,
		Location:              j.Location,
		PostedAt:              j.PostedAt,
		ExperienceRequirement: j.ExperienceRequirement,
		Description:           j.Description,
		Responsibilities:      j.Responsibilities,
		Requirements:          j.Requirements,
		Benefits:              j.Benefits,
		JobTech:               j.JobTech,
		CreatedAt:             j.CreatedAt,
	}
}
