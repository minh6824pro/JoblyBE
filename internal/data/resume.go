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

type Resume struct {
	ID           primitive.ObjectID `bson:"_id"`
	ResumeDetail ResumeDetail       `bson:"resume_detail"`
	Version      int32              `bson:"version"`
	CreatedAt    time.Time          `bson:"created_at"`
}

type ResumeDetail struct {
	Name           string     `bson:"name"`
	Email          string     `bson:"email"`
	Phone          string     `bson:"phone"`
	Summary        string     `bson:"summary"`
	Skills         []string   `bson:"skill"`
	Education      Education  `bson:"education"`
	Experience     Experience `bson:"experience"`
	Certifications []string   `bson:"certifications"`
	Languages      []string   `bson:"languages"`
}

type Education struct {
	Degree         string `bson:"degree"`
	Institution    string `bson:"institution"`
	GraduationYear string `bson:"graduation_year"`
}

type Experience struct {
	Title            string   `bson:"title"`
	Company          string   `bson:"company"`
	Duration         string   `bson:"duration"`
	Responsibilities []string `bson:"responsibilities"`
	Achievements     []string `bson:"achievements"`
}

type resumeRepo struct {
	data *Data
	log  *log.Helper
}

// NewResumeRepo creates a new resume repository
func NewResumeRepo(data *Data, logger log.Logger) biz.ResumeRepo {
	return &resumeRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// CreateResume creates a new resume for a user (adds to user's resume array)
func (r *resumeRepo) CreateResume(ctx context.Context, resume *biz.Resume) (*biz.Resume, error) {
	// Convert UserID to ObjectID
	userObjID, err := primitive.ObjectIDFromHex(resume.UserID)
	if err != nil {
		r.log.Errorf("invalid user ID: %v", err)
		return nil, err
	}

	// Create resume document
	resumeDoc := Resume{
		ID: primitive.NewObjectID(),
		ResumeDetail: ResumeDetail{
			Name:           resume.ResumeDetail.Name,
			Email:          resume.ResumeDetail.Email,
			Phone:          resume.ResumeDetail.Phone,
			Summary:        resume.ResumeDetail.Summary,
			Skills:         resume.ResumeDetail.Skills,
			Education:      r.toEducationDoc(resume.ResumeDetail.Education),
			Experience:     r.toExperienceDoc(resume.ResumeDetail.Experience),
			Certifications: resume.ResumeDetail.Certifications,
			Languages:      resume.ResumeDetail.Languages,
		},
		Version:   1,
		CreatedAt: time.Now(),
	}

	// Push resume to user's resume array
	result, err := r.data.db.Collection(CollectionUser).UpdateOne(
		ctx,
		bson.M{"_id": userObjID},
		bson.M{
			"$push": bson.M{"resume": resumeDoc},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)

	if err != nil {
		r.log.Errorf("failed to create resume: %v", err)
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, mongo.ErrNoDocuments
	}

	resume.ID = resumeDoc.ID.Hex()
	resume.CreatedAt = resumeDoc.CreatedAt
	resume.Version = resumeDoc.Version

	return resume, nil
}

// UpdateResume updates an existing resume in user's resume array
func (r *resumeRepo) UpdateResume(ctx context.Context, resume *biz.Resume) (*biz.Resume, error) {
	userObjID, err := primitive.ObjectIDFromHex(resume.UserID)
	if err != nil {
		r.log.Errorf("invalid user ID: %v", err)
		return nil, err
	}

	resumeObjID, err := primitive.ObjectIDFromHex(resume.ID)
	if err != nil {
		r.log.Errorf("invalid resume ID: %v", err)
		return nil, err
	}

	// Update resume in array using positional operator $
	update := bson.M{
		"$set": bson.M{
			"resume.$.resume_detail": ResumeDetail{
				Name:           resume.ResumeDetail.Name,
				Email:          resume.ResumeDetail.Email,
				Phone:          resume.ResumeDetail.Phone,
				Summary:        resume.ResumeDetail.Summary,
				Skills:         resume.ResumeDetail.Skills,
				Education:      r.toEducationDoc(resume.ResumeDetail.Education),
				Experience:     r.toExperienceDoc(resume.ResumeDetail.Experience),
				Certifications: resume.ResumeDetail.Certifications,
				Languages:      resume.ResumeDetail.Languages,
			},
			"resume.$.version": resume.Version,
			"updated_at":       time.Now(),
		},
	}

	result, err := r.data.db.Collection(CollectionUser).UpdateOne(
		ctx,
		bson.M{
			"_id":        userObjID,
			"resume._id": resumeObjID,
		},
		update,
	)

	if err != nil {
		r.log.Errorf("failed to update resume: %v", err)
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, nil
	}

	return resume, nil
}

// GetResume retrieves a specific resume by ID from user's resume array
func (r *resumeRepo) GetResume(ctx context.Context, id string) (*biz.Resume, error) {
	resumeObjID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.log.Errorf("invalid resume ID: %v", err)
		return nil, err
	}

	// Find user with this resume
	var user User
	err = r.data.db.Collection(CollectionUser).FindOne(
		ctx,
		bson.M{"resume._id": resumeObjID},
	).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.log.Errorf("failed to get resume: %v", err)
		return nil, err
	}

	// Find the specific resume in array
	for _, resumeDoc := range user.Resume {
		if resumeDoc.ID == resumeObjID {
			return r.toBiz(&resumeDoc, user.ID.Hex()), nil
		}
	}

	return nil, nil
}

// ListResumes lists all resumes for a user
func (r *resumeRepo) ListResumes(ctx context.Context, userID string, page, pageSize int32) ([]*biz.Resume, int32, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		r.log.Errorf("invalid user ID: %v", err)
		return nil, 0, err
	}

	// Get user with all resumes
	var user User
	err = r.data.db.Collection(CollectionUser).FindOne(
		ctx,
		bson.M{"_id": userObjID},
	).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []*biz.Resume{}, 0, nil
		}
		r.log.Errorf("failed to get user: %v", err)
		return nil, 0, err
	}

	// Convert all resumes
	total := int32(len(user.Resume))
	resumes := make([]*biz.Resume, 0)

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return []*biz.Resume{}, total, nil
	}
	if end > total {
		end = total
	}

	for i := start; i < end; i++ {
		resumes = append(resumes, r.toBiz(&user.Resume[i], userID))
	}

	return resumes, total, nil
}

// DeleteResume removes a resume from user's resume array
func (r *resumeRepo) DeleteResume(ctx context.Context, id string) error {
	resumeObjID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		r.log.Errorf("invalid resume ID: %v", err)
		return err
	}

	// Pull resume from array
	result, err := r.data.db.Collection(CollectionUser).UpdateOne(
		ctx,
		bson.M{"resume._id": resumeObjID},
		bson.M{
			"$pull": bson.M{"resume": bson.M{"_id": resumeObjID}},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)

	if err != nil {
		r.log.Errorf("failed to delete resume: %v", err)
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// Helper functions
func (r *resumeRepo) toBiz(doc *Resume, userID string) *biz.Resume {
	return &biz.Resume{
		ID:     doc.ID.Hex(),
		UserID: userID,
		ResumeDetail: &biz.ResumeDetail{
			Name:           doc.ResumeDetail.Name,
			Email:          doc.ResumeDetail.Email,
			Phone:          doc.ResumeDetail.Phone,
			Summary:        doc.ResumeDetail.Summary,
			Skills:         doc.ResumeDetail.Skills,
			Education:      r.toEducationBiz(&doc.ResumeDetail.Education),
			Experience:     r.toExperienceBiz(&doc.ResumeDetail.Experience),
			Certifications: doc.ResumeDetail.Certifications,
			Languages:      doc.ResumeDetail.Languages,
		},
		Version:   doc.Version,
		CreatedAt: doc.CreatedAt,
	}
}

func (r *resumeRepo) toEducationBiz(doc *Education) *biz.Education {
	if doc == nil {
		return nil
	}
	return &biz.Education{
		Degree:         doc.Degree,
		Institution:    doc.Institution,
		GraduationYear: doc.GraduationYear,
	}
}

func (r *resumeRepo) toEducationDoc(edu *biz.Education) Education {
	if edu == nil {
		return Education{}
	}
	return Education{
		Degree:         edu.Degree,
		Institution:    edu.Institution,
		GraduationYear: edu.GraduationYear,
	}
}

func (r *resumeRepo) toExperienceBiz(doc *Experience) *biz.Experience {
	if doc == nil {
		return nil
	}
	return &biz.Experience{
		Title:            doc.Title,
		Company:          doc.Company,
		Duration:         doc.Duration,
		Responsibilities: doc.Responsibilities,
		Achievements:     doc.Achievements,
	}
}

func (r *resumeRepo) toExperienceDoc(exp *biz.Experience) Experience {
	if exp == nil {
		return Experience{}
	}
	return Experience{
		Title:            exp.Title,
		Company:          exp.Company,
		Duration:         exp.Duration,
		Responsibilities: exp.Responsibilities,
		Achievements:     exp.Achievements,
	}
}
