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
	Name           string             `bson:"name"`
	Email          string             `bson:"email"`
	Phone          string             `bson:"phone"`
	Summary        string             `bson:"summary"`
	Skills         []string           `bson:"skill"`
	Education      []Education        `bson:"education"`
	Experience     []Experience       `bson:"experience"`
	Projects       []Project          `bson:"projects"`
	Certifications []string           `bson:"certifications"`
	Languages      []string           `bson:"languages"`
	Achievements   []string           `bson:"achievements"`
	Evaluations    []ResumeEvaluation `bson:"evaluations,omitempty"`
}

type Education struct {
	Degree         string `bson:"degree"`
	Institution    string `bson:"institution"`
	GraduationYear string `bson:"graduation_year"`
	Description    string `bson:"description"`
}

type Experience struct {
	Title            string   `bson:"title"`
	Company          string   `bson:"company"`
	Duration         string   `bson:"duration"`
	Description      string   `bson:"description"`
	Responsibilities []string `bson:"responsibilities"`
	Achievements     []string `bson:"achievements"`
}

type Project struct {
	Name         string   `bson:"name"`
	Description  string   `bson:"description"`
	Technologies []string `bson:"technologies"`
	Url          string   `bson:"url"`
	Duration     string   `bson:"duration"`
	Role         string   `bson:"role"`
	Achievements []string `bson:"achievements"`
}

type ResumeScoreBreakdown struct {
	SkillsScore       float64 `bson:"skills_score"`
	ExperienceScore   float64 `bson:"experience_score"`
	EducationScore    float64 `bson:"education_score"`
	CompletenessScore float64 `bson:"completeness_score"`
	JobAlignmentScore float64 `bson:"job_alignment_score"`
	PresentationScore float64 `bson:"presentation_score"`
}

type ResumeCVEdit struct {
	ID             string  `bson:"id"`
	FieldPath      string  `bson:"field_path"`
	Action         string  `bson:"action"`
	CurrentValue   string  `bson:"current_value"`
	SuggestedValue string  `bson:"suggested_value"`
	Reason         string  `bson:"reason"`
	Priority       string  `bson:"priority"`
	ImpactScore    float64 `bson:"impact_score"`
	Status         string  `bson:"status,omitempty"`
}

type ResumeEvaluation struct {
	CVName          string               `bson:"cv_name"`
	OverallScore    float64              `bson:"overall_score"`
	Grade           string               `bson:"grade"`
	ScoreBreakdown  ResumeScoreBreakdown `bson:"score_breakdown"`
	Strengths       []string             `bson:"strengths"`
	Weaknesses      []string             `bson:"weaknesses"`
	Recommendations []string             `bson:"recommendations"`
	CVEdits         []ResumeCVEdit       `bson:"cv_edits"`
	JobsAnalyzed    int                  `bson:"jobs_analyzed"`
	EvaluatedAt     time.Time            `bson:"evaluated_at"`
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
			Education:      r.toEducationDocsArray(resume.ResumeDetail.Education),
			Experience:     r.toExperienceDocsArray(resume.ResumeDetail.Experience),
			Projects:       r.toProjectDocsArray(resume.ResumeDetail.Projects),
			Certifications: resume.ResumeDetail.Certifications,
			Languages:      resume.ResumeDetail.Languages,
			Achievements:   resume.ResumeDetail.Achievements,
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
				Education:      r.toEducationDocsArray(resume.ResumeDetail.Education),
				Experience:     r.toExperienceDocsArray(resume.ResumeDetail.Experience),
				Projects:       r.toProjectDocsArray(resume.ResumeDetail.Projects),
				Certifications: resume.ResumeDetail.Certifications,
				Languages:      resume.ResumeDetail.Languages,
				Achievements:   resume.ResumeDetail.Achievements,
				Evaluations:    r.toEvaluationsArray(resume.ResumeDetail.Evaluations),
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
			Education:      r.toEducationBizArray(doc.ResumeDetail.Education),
			Experience:     r.toExperienceBizArray(doc.ResumeDetail.Experience),
			Projects:       r.toProjectBizArray(doc.ResumeDetail.Projects),
			Certifications: doc.ResumeDetail.Certifications,
			Languages:      doc.ResumeDetail.Languages,
			Achievements:   doc.ResumeDetail.Achievements,
			Evaluations:    r.toEvaluationsBizArray(doc.ResumeDetail.Evaluations),
		},
		Version:   doc.Version,
		CreatedAt: doc.CreatedAt,
	}
}

func (r *resumeRepo) toEducationBizArray(docs []Education) []*biz.Education {
	if docs == nil {
		return nil
	}
	result := make([]*biz.Education, 0, len(docs))
	for i := range docs {
		result = append(result, &biz.Education{
			Degree:         docs[i].Degree,
			Institution:    docs[i].Institution,
			GraduationYear: docs[i].GraduationYear,
			Description:    docs[i].Description,
		})
	}
	return result
}

func (r *resumeRepo) toEducationDocsArray(edus []*biz.Education) []Education {
	if edus == nil {
		return nil
	}
	result := make([]Education, 0, len(edus))
	for _, edu := range edus {
		if edu != nil {
			result = append(result, Education{
				Degree:         edu.Degree,
				Institution:    edu.Institution,
				GraduationYear: edu.GraduationYear,
				Description:    edu.Description,
			})
		}
	}
	return result
}

func (r *resumeRepo) toExperienceBizArray(docs []Experience) []*biz.Experience {
	if docs == nil {
		return nil
	}
	result := make([]*biz.Experience, 0, len(docs))
	for i := range docs {
		result = append(result, &biz.Experience{
			Title:            docs[i].Title,
			Company:          docs[i].Company,
			Duration:         docs[i].Duration,
			Description:      docs[i].Description,
			Responsibilities: docs[i].Responsibilities,
			Achievements:     docs[i].Achievements,
		})
	}
	return result
}

func (r *resumeRepo) toExperienceDocsArray(exps []*biz.Experience) []Experience {
	if exps == nil {
		return nil
	}
	result := make([]Experience, 0, len(exps))
	for _, exp := range exps {
		if exp != nil {
			result = append(result, Experience{
				Title:            exp.Title,
				Company:          exp.Company,
				Duration:         exp.Duration,
				Description:      exp.Description,
				Responsibilities: exp.Responsibilities,
				Achievements:     exp.Achievements,
			})
		}
	}
	return result
}

func (r *resumeRepo) toProjectBizArray(docs []Project) []*biz.Project {
	if docs == nil {
		return nil
	}
	result := make([]*biz.Project, 0, len(docs))
	for i := range docs {
		result = append(result, &biz.Project{
			Name:         docs[i].Name,
			Description:  docs[i].Description,
			Technologies: docs[i].Technologies,
			Url:          docs[i].Url,
			Duration:     docs[i].Duration,
			Role:         docs[i].Role,
			Achievements: docs[i].Achievements,
		})
	}
	return result
}

func (r *resumeRepo) toProjectDocsArray(projects []*biz.Project) []Project {
	if projects == nil {
		return nil
	}
	result := make([]Project, 0, len(projects))
	for _, proj := range projects {
		if proj != nil {
			result = append(result, Project{
				Name:         proj.Name,
				Description:  proj.Description,
				Technologies: proj.Technologies,
				Url:          proj.Url,
				Duration:     proj.Duration,
				Role:         proj.Role,
				Achievements: proj.Achievements,
			})
		}
	}
	return result
}

func (r *resumeRepo) toEvaluationsArray(evals []*biz.ResumeEvaluation) []ResumeEvaluation {
	if evals == nil {
		return []ResumeEvaluation{}
	}
	result := make([]ResumeEvaluation, 0, len(evals))
	for _, eval := range evals {
		if eval != nil {
			dataEval := ResumeEvaluation{
				CVName:          eval.CVName,
				OverallScore:    eval.OverallScore,
				Grade:           eval.Grade,
				Strengths:       eval.Strengths,
				Weaknesses:      eval.Weaknesses,
				Recommendations: eval.Recommendations,
				JobsAnalyzed:    eval.JobsAnalyzed,
				EvaluatedAt:     eval.EvaluatedAt,
			}

			// Convert ScoreBreakdown
			if eval.ScoreBreakdown != nil {
				dataEval.ScoreBreakdown = ResumeScoreBreakdown{
					SkillsScore:       eval.ScoreBreakdown.SkillsScore,
					ExperienceScore:   eval.ScoreBreakdown.ExperienceScore,
					EducationScore:    eval.ScoreBreakdown.EducationScore,
					CompletenessScore: eval.ScoreBreakdown.CompletenessScore,
					JobAlignmentScore: eval.ScoreBreakdown.JobAlignmentScore,
					PresentationScore: eval.ScoreBreakdown.PresentationScore,
				}
			}

			// Convert CVEdits
			if len(eval.CVEdits) > 0 {
				dataEval.CVEdits = make([]ResumeCVEdit, 0, len(eval.CVEdits))
				for _, edit := range eval.CVEdits {
					if edit != nil {
						dataEval.CVEdits = append(dataEval.CVEdits, ResumeCVEdit{
							ID:             edit.ID,
							FieldPath:      edit.FieldPath,
							Action:         edit.Action,
							CurrentValue:   edit.CurrentValue,
							SuggestedValue: edit.SuggestedValue,
							Reason:         edit.Reason,
							Priority:       edit.Priority,
							ImpactScore:    edit.ImpactScore,
							Status:         edit.Status,
						})
					}
				}
			}

			result = append(result, dataEval)
		}
	}
	return result
}

func (r *resumeRepo) toEvaluationsBizArray(docs []ResumeEvaluation) []*biz.ResumeEvaluation {
	if docs == nil {
		return nil
	}
	result := make([]*biz.ResumeEvaluation, 0, len(docs))
	for i := range docs {
		bizEval := &biz.ResumeEvaluation{
			CVName:          docs[i].CVName,
			OverallScore:    docs[i].OverallScore,
			Grade:           docs[i].Grade,
			Strengths:       docs[i].Strengths,
			Weaknesses:      docs[i].Weaknesses,
			Recommendations: docs[i].Recommendations,
			JobsAnalyzed:    docs[i].JobsAnalyzed,
			EvaluatedAt:     docs[i].EvaluatedAt,
		}

		// Convert ScoreBreakdown
		bizEval.ScoreBreakdown = &biz.ResumeScoreBreakdown{
			SkillsScore:       docs[i].ScoreBreakdown.SkillsScore,
			ExperienceScore:   docs[i].ScoreBreakdown.ExperienceScore,
			EducationScore:    docs[i].ScoreBreakdown.EducationScore,
			CompletenessScore: docs[i].ScoreBreakdown.CompletenessScore,
			JobAlignmentScore: docs[i].ScoreBreakdown.JobAlignmentScore,
			PresentationScore: docs[i].ScoreBreakdown.PresentationScore,
		}

		// Convert CVEdits
		if len(docs[i].CVEdits) > 0 {
			bizEval.CVEdits = make([]*biz.ResumeCVEdit, 0, len(docs[i].CVEdits))
			for j := range docs[i].CVEdits {
				bizEval.CVEdits = append(bizEval.CVEdits, &biz.ResumeCVEdit{
					ID:             docs[i].CVEdits[j].ID,
					FieldPath:      docs[i].CVEdits[j].FieldPath,
					Action:         docs[i].CVEdits[j].Action,
					CurrentValue:   docs[i].CVEdits[j].CurrentValue,
					SuggestedValue: docs[i].CVEdits[j].SuggestedValue,
					Reason:         docs[i].CVEdits[j].Reason,
					Priority:       docs[i].CVEdits[j].Priority,
					ImpactScore:    docs[i].CVEdits[j].ImpactScore,
					Status:         docs[i].CVEdits[j].Status,
				})
			}
		}

		result = append(result, bizEval)
	}
	return result
}
