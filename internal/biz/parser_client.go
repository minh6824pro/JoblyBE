package biz

import (
	"context"
	"fmt"

	"JobblyBE/pkg/configx"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/imroc/req/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ParserClient struct {
	client     *req.Client
	trackingUC *UserTrackingUseCase
	jobRepo    JobPostingRepo
	log        *log.Helper
}

type ErrorResponse struct {
	Code     int                    `json:"code"`
	Reason   string                 `json:"reason"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata"`
}

// EvaluateCV represents the CV section of the evaluate request
type EvaluateCV struct {
	Name           string                `json:"name"`
	Email          string                `json:"email"`
	Phone          string                `json:"phone"`
	Summary        string                `json:"summary"`
	Skills         []string              `json:"skills"`
	Education      []*EvaluateEducation  `json:"education"`
	Experience     []*EvaluateExperience `json:"experience"`
	Projects       []*EvaluateProject    `json:"projects"`
	Certifications []string              `json:"certifications"`
	Languages      []string              `json:"languages"`
	Achievements   []string              `json:"achievements"`
}

type EvaluateEducation struct {
	Degree         string  `json:"degree"`
	Institution    string  `json:"institution"`
	GraduationYear int     `json:"graduation_year"`
	Description    string  `json:"description"`
	GPA            float64 `json:"gpa"`
}

type EvaluateExperience struct {
	Title            string   `json:"title"`
	Company          string   `json:"company"`
	Duration         string   `json:"duration"`
	Description      string   `json:"description"`
	Responsibilities []string `json:"responsibilities"`
	Achievements     []string `json:"achievements"`
}

type EvaluateProject struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies"`
	URL          string   `json:"url"`
	Duration     string   `json:"duration"`
	Role         string   `json:"role"`
	Achievements []string `json:"achievements"`
}

type JobDescription struct {
	Title                   string   `json:"title"`
	Company                 string   `json:"company"`
	Requirements            []string `json:"requirements"`
	Responsibilities        []string `json:"responsibilities"`
	PreferredQualifications []string `json:"preferred_qualifications"`
	RequiredSkills          []string `json:"required_skills"`
}

type InteractionHistory struct {
	JobDescriptions  []*JobDescription `json:"job_descriptions"`
	InteractionCount int               `json:"interaction_count"`
}

type EvaluateRequest struct {
	CV                 *EvaluateCV         `json:"cv"`
	InteractionHistory *InteractionHistory `json:"interaction_history"`
}

type ScoreBreakdown struct {
	SkillsScore       float64 `json:"skills_score"`
	ExperienceScore   float64 `json:"experience_score"`
	EducationScore    float64 `json:"education_score"`
	CompletenessScore float64 `json:"completeness_score"`
	JobAlignmentScore float64 `json:"job_alignment_score"`
	PresentationScore float64 `json:"presentation_score"`
}

type CVEdit struct {
	FieldPath      string  `json:"field_path"`
	Action         string  `json:"action"`
	CurrentValue   string  `json:"current_value"`
	SuggestedValue string  `json:"suggested_value"`
	Reason         string  `json:"reason"`
	Priority       string  `json:"priority"`
	ImpactScore    float64 `json:"impact_score"`
}

type EvaluateResponse struct {
	Success         bool            `json:"success"`
	CVName          string          `json:"cv_name"`
	OverallScore    float64         `json:"overall_score"`
	Grade           string          `json:"grade"`
	ScoreBreakdown  *ScoreBreakdown `json:"score_breakdown"`
	Strengths       []string        `json:"strengths"`
	Weaknesses      []string        `json:"weaknesses"`
	Recommendations []string        `json:"recommendations"`
	CVEdits         []*CVEdit       `json:"cv_edits"`
	JobsAnalyzed    int             `json:"jobs_analyzed"`
	Deterministic   bool            `json:"deterministic"`
	Error           string          `json:"error"`
}

func NewParserClient(trackingUC *UserTrackingUseCase, jobRepo JobPostingRepo, logger log.Logger) *ParserClient {
	addr := configx.GetEnvOrString("RESUME_PARSER_URL", "")
	fmt.Println("[Init NewParserClient] Parser service address: ", addr)
	return &ParserClient{
		client: req.C().
			SetBaseURL(addr).
			SetResultStateCheckFunc(func(resp *req.Response) req.ResultState {
				if resp.StatusCode < 200 || resp.StatusCode >= 400 {
					return req.ErrorState
				}
				return req.SuccessState
			}),
		trackingUC: trackingUC,
		jobRepo:    jobRepo,
		log:        log.NewHelper(logger),
	}
}

// EvaluateResume calls the parser service to evaluate a resume against interaction history
func (pc *ParserClient) EvaluateResume(ctx context.Context, userID string, resumeDetail *ResumeDetail) (*EvaluateResponse, error) {
	// Convert userID to ObjectID
	userIDObject, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get top 2 viewed jobs in the last week
	topJobs, err := pc.trackingUC.UserTrackingRepo.GetTopViewedJobsInLastWeek(ctx, userIDObject, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to get top viewed jobs: %w", err)
	}

	// Build job descriptions from tracking data
	jobDescriptions := make([]*JobDescription, 0, len(topJobs))
	for _, tracking := range topJobs {
		// Get job details
		jobID := tracking.JobID.Hex()
		job, err := pc.jobRepo.GetJobPosting(ctx, jobID)
		if err != nil {
			pc.log.Warnf("failed to get job posting %s: %v", jobID, err)
			continue
		}

		// Extract requirements and responsibilities
		requirements := []string{}
		if job.Requirements != "" {
			requirements = append(requirements, job.Requirements)
		}

		responsibilities := []string{}
		if job.Responsibilities != "" {
			responsibilities = append(responsibilities, job.Responsibilities)
		}

		companyName := ""
		if job.Company != nil {
			companyName = job.Company.Name
		}

		jobDesc := &JobDescription{
			Title:                   job.Title,
			Company:                 companyName,
			Requirements:            requirements,
			Responsibilities:        responsibilities,
			PreferredQualifications: []string{},
			RequiredSkills:          job.JobTech,
		}

		jobDescriptions = append(jobDescriptions, jobDesc)
	}

	// Convert resume detail to evaluate CV format
	evaluateCV := pc.convertResumeToEvaluateCV(resumeDetail)

	// Build request
	request := &EvaluateRequest{
		CV: evaluateCV,
		InteractionHistory: &InteractionHistory{
			JobDescriptions:  jobDescriptions,
			InteractionCount: len(jobDescriptions),
		},
	}

	// Log request for debugging
	pc.log.Infof("Calling evaluate API with %d job descriptions", len(jobDescriptions))

	// Call parser service
	var response EvaluateResponse
	var errorResp ErrorResponse
	resp, err := pc.client.R().
		SetContext(ctx).
		SetBody(request).
		SetSuccessResult(&response).
		SetErrorResult(&errorResp).
		Post("/evaluate")

	if err != nil {
		pc.log.Errorf("Failed to call evaluate API: %v", err)
		return nil, fmt.Errorf("failed to call evaluate API: %w", err)
	}

	if !resp.IsSuccessState() {
		// Log detailed error
		pc.log.Errorf("Evaluate API error - Status: %d, Error: %+v, Body: %s",
			resp.StatusCode, errorResp, string(resp.Bytes()))
		return nil, fmt.Errorf("evaluate API returned error: status %d, reason: %s, message: %s",
			resp.StatusCode, errorResp.Reason, errorResp.Message)
	}

	pc.log.Infof("Evaluate API success - Score: %.2f, Grade: %s", response.OverallScore, response.Grade)
	return &response, nil
}

// convertResumeToEvaluateCV converts ResumeDetail to EvaluateCV format
func (pc *ParserClient) convertResumeToEvaluateCV(detail *ResumeDetail) *EvaluateCV {
	cv := &EvaluateCV{
		Name:           detail.Name,
		Email:          detail.Email,
		Phone:          detail.Phone,
		Summary:        detail.Summary,
		Skills:         []string{},
		Education:      []*EvaluateEducation{},
		Experience:     []*EvaluateExperience{},
		Projects:       []*EvaluateProject{},
		Certifications: []string{},
		Languages:      []string{},
		Achievements:   []string{},
	}

	// Initialize with existing data or empty arrays
	if detail.Skills != nil {
		cv.Skills = detail.Skills
	}
	if detail.Certifications != nil {
		cv.Certifications = detail.Certifications
	}
	if detail.Languages != nil {
		cv.Languages = detail.Languages
	}
	if detail.Achievements != nil {
		cv.Achievements = detail.Achievements
	}

	// Convert Education
	if len(detail.Education) > 0 {
		cv.Education = make([]*EvaluateEducation, 0, len(detail.Education))
		for _, edu := range detail.Education {
			// Parse graduation year from string
			gradYear := 0
			fmt.Sscanf(edu.GraduationYear, "%d", &gradYear)

			cv.Education = append(cv.Education, &EvaluateEducation{
				Degree:         edu.Degree,
				Institution:    edu.Institution,
				GraduationYear: gradYear,
				Description:    edu.Description,
				GPA:            0, // GPA not available in current structure
			})
		}
	}

	// Convert Experience
	if len(detail.Experience) > 0 {
		cv.Experience = make([]*EvaluateExperience, 0, len(detail.Experience))
		for _, exp := range detail.Experience {
			cv.Experience = append(cv.Experience, &EvaluateExperience{
				Title:            exp.Title,
				Company:          exp.Company,
				Duration:         exp.Duration,
				Description:      exp.Description,
				Responsibilities: exp.Responsibilities,
				Achievements:     exp.Achievements,
			})
		}
	}

	// Convert Projects
	if len(detail.Projects) > 0 {
		cv.Projects = make([]*EvaluateProject, 0, len(detail.Projects))
		for _, proj := range detail.Projects {
			cv.Projects = append(cv.Projects, &EvaluateProject{
				Name:         proj.Name,
				Description:  proj.Description,
				Technologies: proj.Technologies,
				URL:          proj.Url,
				Duration:     proj.Duration,
				Role:         proj.Role,
				Achievements: proj.Achievements,
			})
		}
	}

	return cv
}
