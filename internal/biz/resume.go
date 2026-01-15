package biz

import (
	"JobblyBE/pkg/configx"
	"JobblyBE/pkg/kafkax"
	"JobblyBE/pkg/openai"
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Resume represents a user's resume
type Resume struct {
	ID           string
	UserID       string
	ResumeDetail *ResumeDetail
	Version      int32
	CreatedAt    time.Time
}

type ResumeDetail struct {
	Name           string
	Email          string
	Phone          string
	Summary        string
	Skills         []string
	Education      []*Education
	Experience     []*Experience
	Projects       []*Project
	Certifications []string
	Languages      []string
	Achievements   []string
	Evaluations    []*ResumeEvaluation
}

type Education struct {
	Degree         string
	Institution    string
	GraduationYear string
	Description    string
}

type Experience struct {
	Title            string
	Company          string
	Duration         string
	Description      string
	Responsibilities []string
	Achievements     []string
}

type Project struct {
	Name         string
	Description  string
	Technologies []string
	Url          string
	Duration     string
	Role         string
	Achievements []string
}

type ResumeScoreBreakdown struct {
	SkillsScore       float64 `json:"skills_score"`
	ExperienceScore   float64 `json:"experience_score"`
	EducationScore    float64 `json:"education_score"`
	CompletenessScore float64 `json:"completeness_score"`
	JobAlignmentScore float64 `json:"job_alignment_score"`
	PresentationScore float64 `json:"presentation_score"`
}

type ResumeCVEdit struct {
	ID             string  `json:"id"`
	FieldPath      string  `json:"field_path"`
	Action         string  `json:"action"`
	CurrentValue   string  `json:"current_value"`
	SuggestedValue string  `json:"suggested_value"`
	Reason         string  `json:"reason"`
	Priority       string  `json:"priority"`
	ImpactScore    float64 `json:"impact_score"`
	Status         string  `json:"status"` // empty/null: new, "rejected", "accepted"
}

type ResumeEvaluation struct {
	CVName          string                `json:"cv_name"`
	OverallScore    float64               `json:"overall_score"`
	Grade           string                `json:"grade"`
	ScoreBreakdown  *ResumeScoreBreakdown `json:"score_breakdown"`
	Strengths       []string              `json:"strengths"`
	Weaknesses      []string              `json:"weaknesses"`
	Recommendations []string              `json:"recommendations"`
	CVEdits         []*ResumeCVEdit       `json:"cv_edits"`
	JobsAnalyzed    int                   `json:"jobs_analyzed"`
	EvaluatedAt     time.Time             `json:"evaluated_at"`
	Type            string                `json:"type"`      // "auto" (default) or "manual"
	JobID           string                `json:"job_id"`    // Job ID used for manual evaluation
	JobTitle        string                `json:"job_title"` // Job title for display
}

type ScoreWithJD struct {
	JobID          string                `json:"job_id"`
	OverallScore   float64               `json:"overall_score"`
	ScoreBreakdown *ResumeScoreBreakdown `json:"score_breakdown"`
	MissingSkill   []string              `json:"missing_skill"`
}

// ResumeRepo is the interface for resume repository
type ResumeRepo interface {
	CreateResume(ctx context.Context, resume *Resume) (*Resume, error)
	UpdateResume(ctx context.Context, resume *Resume) (*Resume, error)
	GetResume(ctx context.Context, id string) (*Resume, error)
	ListResumes(ctx context.Context, userID string, page, pageSize int32) ([]*Resume, int32, error)
	DeleteResume(ctx context.Context, id string) error
}

// ResumeUseCase is the use case for resume operations
type ResumeUseCase struct {
	repo         ResumeRepo
	userRepo     UserRepo
	jobRepo      JobPostingRepo
	trackingUC   *UserTrackingUseCase
	parserClient *ParserClient
	producer     *kafkax.Producer
	openAIKey    string
	log          *log.Helper
}

// NewResumeUseCase creates a new resume use case
func NewResumeUseCase(repo ResumeRepo, userRepo UserRepo, jobRepo JobPostingRepo, trackingUC *UserTrackingUseCase, parserClient *ParserClient, producer *kafkax.Producer, logger log.Logger) *ResumeUseCase {
	return &ResumeUseCase{
		repo:         repo,
		userRepo:     userRepo,
		jobRepo:      jobRepo,
		trackingUC:   trackingUC,
		parserClient: parserClient,
		producer:     producer,
		openAIKey:    configx.GetEnvOrString("OPENAI_API_KEY", ""),
		log:          log.NewHelper(logger),
	}
}

// CreateResume creates a new resume
func (uc *ResumeUseCase) CreateResume(ctx context.Context, resume *Resume) (*Resume, error) {
	// Validate resume
	if err := uc.validateResume(resume); err != nil {
		return nil, err
	}

	// Set timestamp and version
	resume.CreatedAt = time.Now()
	resume.Version = 1

	createdResume, err := uc.repo.CreateResume(ctx, resume)
	if err != nil {
		return nil, err
	}

	// Push evaluate event to Kafka
	if uc.producer != nil {
		evaluateMsg := map[string]interface{}{
			"resume_id": createdResume.ID,
			"user_id":   createdResume.UserID,
		}
		if err := uc.producer.SendMessage(ctx, "evaluate", createdResume.ID, evaluateMsg); err != nil {
			uc.log.Errorf("Failed to push evaluate message after create: %v", err)
			// Don't fail the request, just log
		} else {
			uc.log.Infof("Pushed evaluate message for new resume: %s", createdResume.ID)
		}
	}

	return createdResume, nil
}

// UpdateResume updates an existing resume
func (uc *ResumeUseCase) UpdateResume(ctx context.Context, resume *Resume) (*Resume, error) {
	// Validate resume
	if err := uc.validateResume(resume); err != nil {
		return nil, err
	}

	// Check if resume exists
	existing, err := uc.repo.GetResume(ctx, resume.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrResumeNotFound
	}

	// Check ownership
	if existing.UserID != resume.UserID {
		return nil, ErrUnauthorized
	}

	// Increment version and preserve creation time
	resume.Version = existing.Version + 1
	resume.CreatedAt = existing.CreatedAt

	updatedResume, err := uc.repo.UpdateResume(ctx, resume)
	if err != nil {
		return nil, err
	}

	// Push evaluate event to Kafka
	if uc.producer != nil {
		evaluateMsg := map[string]interface{}{
			"resume_id": updatedResume.ID,
			"user_id":   updatedResume.UserID,
		}
		if err := uc.producer.SendMessage(ctx, "evaluate", updatedResume.ID, evaluateMsg); err != nil {
			uc.log.Errorf("Failed to push evaluate message after update: %v", err)
			// Don't fail the request, just log
		} else {
			uc.log.Infof("Pushed evaluate message for updated resume: %s", updatedResume.ID)
		}
	}

	return updatedResume, nil
}

// GetResume retrieves a resume by ID
func (uc *ResumeUseCase) GetResume(ctx context.Context, id, userID string) (*Resume, error) {
	resume, err := uc.repo.GetResume(ctx, id)
	if err != nil {
		return nil, err
	}
	if resume == nil {
		return nil, ErrResumeNotFound
	}

	// Check ownership
	if resume.UserID != userID {
		return nil, ErrUnauthorized
	}

	return resume, nil
}

// ListResumes lists all resumes for a user
func (uc *ResumeUseCase) ListResumes(ctx context.Context, userID string, page, pageSize int32) ([]*Resume, int32, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	return uc.repo.ListResumes(ctx, userID, page, pageSize)
}

// DeleteResume deletes a resume
func (uc *ResumeUseCase) DeleteResume(ctx context.Context, id, userID string) error {
	// Check if resume exists
	existing, err := uc.repo.GetResume(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrResumeNotFound
	}

	// Check ownership
	if existing.UserID != userID {
		return ErrUnauthorized
	}

	return uc.repo.DeleteResume(ctx, id)
}

// validateResume validates resume data
func (uc *ResumeUseCase) validateResume(resume *Resume) error {
	if resume.ResumeDetail == nil {
		return ErrInvalidResume
	}

	if resume.ResumeDetail.Name == "" {
		return ErrInvalidResume
	}

	if resume.ResumeDetail.Email == "" {
		return ErrInvalidResume
	}

	return nil
}

// GenerateCVDescription generates CV content using ChatGPT based on field tag and CV data
func (uc *ResumeUseCase) GenerateCVDescription(ctx context.Context, resumeDetail *ResumeDetail, userID, fieldTag, currentInput string) (string, error) {
	var prompt string

	// Try to get most viewed job for context
	job, err := uc.trackingUC.GetMostViewedJobByUser(ctx, userID)
	if err != nil {
		// No tracking found, build prompt without job context
		uc.log.Info("No job tracking found for user, generating content based on CV only")
		prompt = uc.buildPromptByFieldTag(fieldTag, currentInput, resumeDetail, nil)
	} else {
		// Build prompt with job context
		prompt = uc.buildPromptByFieldTag(fieldTag, currentInput, resumeDetail, job)
	}

	// Call ChatGPT API
	content, err := uc.callChatGPT(ctx, prompt)
	if err != nil {
		return "", err
	}

	return content, nil
}

// buildPromptByFieldTag builds prompt based on the field tag
func (uc *ResumeUseCase) buildPromptByFieldTag(fieldTag, currentInput string, cv *ResumeDetail, job *JobPosting) string {
	var prompt string

	// Base CV information
	cvInfo := uc.buildCVInfoSection(cv)

	// Job context if available
	jobContext := ""
	if job != nil {
		jobContext = uc.buildJobContextSection(job)
	}

	switch fieldTag {
	case "summary", "description", "objective":
		prompt = uc.buildSummaryPrompt(cvInfo, jobContext, currentInput)
	case "experience_description":
		prompt = uc.buildExperienceDescriptionPrompt(cvInfo, jobContext, currentInput)
	case "education_description":
		prompt = uc.buildEducationDescriptionPrompt(cvInfo, jobContext, currentInput)
	case "project_description":
		prompt = uc.buildProjectDescriptionPrompt(cvInfo, jobContext, currentInput)
	default:
		// Unsupported field tag
		return ""
	}

	return prompt
}

// buildCVInfoSection creates CV information section
func (uc *ResumeUseCase) buildCVInfoSection(cv *ResumeDetail) string {
	info := "CANDIDATE INFORMATION:\n"
	info += "Name: " + cv.Name + "\n"

	if cv.Email != "" {
		info += "Email: " + cv.Email + "\n"
	}

	if cv.Phone != "" {
		info += "Phone: " + cv.Phone + "\n"
	}

	if cv.Summary != "" {
		info += "Summary: " + cv.Summary + "\n"
	}

	if len(cv.Skills) > 0 {
		info += "Skills: "
		for i, skill := range cv.Skills {
			if i > 0 {
				info += ", "
			}
			info += skill
		}
		info += "\n"
	}

	if len(cv.Education) > 0 {
		info += "Education:\n"
		for _, edu := range cv.Education {
			info += "- " + edu.Degree + " at " + edu.Institution
			if edu.GraduationYear != "" {
				info += " (" + edu.GraduationYear + ")"
			}
			info += "\n"
		}
	}

	if len(cv.Experience) > 0 {
		info += "Experience:\n"
		for _, exp := range cv.Experience {
			info += "- " + exp.Title + " at " + exp.Company
			if exp.Duration != "" {
				info += " (" + exp.Duration + ")"
			}
			info += "\n"
			if len(exp.Responsibilities) > 0 {
				info += "  Responsibilities:\n"
				for _, resp := range exp.Responsibilities {
					info += "  * " + resp + "\n"
				}
			}
			if len(exp.Achievements) > 0 {
				info += "  Achievements:\n"
				for _, ach := range exp.Achievements {
					info += "  * " + ach + "\n"
				}
			}
		}
	}

	if len(cv.Certifications) > 0 {
		info += "Certifications: "
		for i, cert := range cv.Certifications {
			if i > 0 {
				info += ", "
			}
			info += cert
		}
		info += "\n"
	}

	if len(cv.Languages) > 0 {
		info += "Languages: "
		for i, lang := range cv.Languages {
			if i > 0 {
				info += ", "
			}
			info += lang
		}
		info += "\n"
	}

	if len(cv.Achievements) > 0 {
		info += "Achievements:\n"
		for _, ach := range cv.Achievements {
			info += "- " + ach + "\n"
		}
	}

	return info
}

// buildJobContextSection creates job context section
func (uc *ResumeUseCase) buildJobContextSection(job *JobPosting) string {
	context := "\nTARGET JOB CONTEXT:\n"
	context += "Position: " + job.Title + "\n"

	if job.Company != nil {
		context += "Company: " + job.Company.Name + "\n"
	}

	if job.Description != "" {
		context += "Job Description: " + job.Description + "\n"
	}

	if job.Requirements != "" {
		context += "Requirements: " + job.Requirements + "\n"
	}

	return context
}

// buildSummaryPrompt builds prompt for summary/description/objective
func (uc *ResumeUseCase) buildSummaryPrompt(cvInfo, jobContext, currentInput string) string {
	prompt := "Write a professional CV summary/description in FIRST PERSON perspective (using 'I', 'my', 'me').\n\n"
	prompt += cvInfo
	prompt += jobContext

	if currentInput != "" {
		prompt += "\nCurrent Summary: " + currentInput + "\n"
		prompt += "\nImprove and enhance the current summary based on the candidate's information"
		if jobContext != "" {
			prompt += " and target job requirements"
		}
		prompt += ".\n"
	}

	prompt += "\nWrite a concise, compelling 2-3 paragraph summary that:\n"
	prompt += "- Highlights key strengths and expertise\n"
	prompt += "- Emphasizes relevant experience and skills\n"
	if jobContext != "" {
		prompt += "- Shows alignment with the target position\n"
	}
	prompt += "- Uses first person (I am, I have, My experience, etc.)\n"
	prompt += "\nCRITICAL LANGUAGE REQUIREMENT:\n"
	prompt += "Carefully analyze the language used in the candidate's CV information above.\n"
	prompt += "If the CV contains Vietnamese text (e.g., 'Đại học', 'Công ty', 'Kinh nghiệm', Vietnamese names/addresses), write your response in Vietnamese.\n"
	prompt += "If the CV is in English, write your response in English.\n"
	prompt += "Match the language of the CV exactly - do not translate or mix languages.\n"
	prompt += "\nProvide only the summary text without any additional explanation."

	return prompt
}

// buildExperienceDescriptionPrompt builds prompt for experience description
func (uc *ResumeUseCase) buildExperienceDescriptionPrompt(cvInfo, jobContext, currentInput string) string {
	prompt := "You are an expert CV/resume writer. Generate a professional experience description paragraph for a CV.\n\n"
	prompt += cvInfo
	prompt += jobContext

	if currentInput != "" {
		prompt += "\nCurrent Description: " + currentInput + "\n"
		prompt += "\nImprove and enhance the current experience description based on the candidate's information"
		if jobContext != "" {
			prompt += " and target job requirements"
		}
		prompt += ".\n"
	}

	prompt += "\nWrite a SHORT, CONCISE description (2-3 sentences MAX) that:\n"
	prompt += "- Summarizes the key responsibility and main achievement in this role\n"
	prompt += "- Uses action verbs without first person (no 'I', 'my', 'me')\n"
	prompt += "- Includes one key metric or result if available\n"
	if jobContext != "" {
		prompt += "- Emphasizes skills relevant to the target position\n"
	}
	prompt += "\nCRITICAL LANGUAGE REQUIREMENT:\n"
	prompt += "Carefully analyze the language used in the candidate's CV information above.\n"
	prompt += "If the CV contains Vietnamese text (e.g., 'Đại học', 'Công ty', 'Kinh nghiệm', Vietnamese names/addresses), write your response in Vietnamese.\n"
	prompt += "If the CV is in English, write your response in English.\n"
	prompt += "Match the language of the CV exactly - do not translate or mix languages.\n"
	prompt += "\nProvide only the paragraph text without any headers, labels, or additional explanation."

	return prompt
}

// buildEducationDescriptionPrompt builds prompt for education description
func (uc *ResumeUseCase) buildEducationDescriptionPrompt(cvInfo, jobContext, currentInput string) string {
	prompt := "Write professional education description in FIRST PERSON perspective (using 'I', 'my', 'me').\n\n"
	prompt += cvInfo
	prompt += jobContext

	if currentInput != "" {
		prompt += "\nCurrent Description: " + currentInput + "\n"
		prompt += "\nImprove and enhance the current education description based on the candidate's information"
		if jobContext != "" {
			prompt += " and target job requirements"
		}
		prompt += ".\n"
	}

	prompt += "\nWrite a concise, compelling 1-2 paragraph description that:\n"
	prompt += "- Highlights relevant coursework and academic achievements\n"
	prompt += "- Emphasizes skills and knowledge gained\n"
	prompt += "- Shows relevance to career goals"
	if jobContext != "" {
		prompt += " and target position"
	}
	prompt += "\n"
	prompt += "- Uses first person (I studied, I specialized in, My coursework included, etc.)\n"
	prompt += "\nCRITICAL LANGUAGE REQUIREMENT:\n"
	prompt += "Carefully analyze the language used in the candidate's CV information above.\n"
	prompt += "If the CV contains Vietnamese text (e.g., 'Đại học', 'Công ty', 'Kinh nghiệm', Vietnamese names/addresses), write your response in Vietnamese.\n"
	prompt += "If the CV is in English, write your response in English.\n"
	prompt += "Match the language of the CV exactly - do not translate or mix languages.\n"
	prompt += "\nProvide only the description text without any additional explanation."

	return prompt
}

// buildProjectDescriptionPrompt builds prompt for project description
func (uc *ResumeUseCase) buildProjectDescriptionPrompt(cvInfo, jobContext, currentInput string) string {
	prompt := "You are an expert CV/resume writer. Generate a professional project description paragraph for a CV.\n\n"
	prompt += cvInfo
	prompt += jobContext

	if currentInput != "" {
		prompt += "\nCurrent Description: " + currentInput + "\n"
		prompt += "\nImprove and enhance the current project description based on the candidate's information"
		if jobContext != "" {
			prompt += " and target job requirements"
		}
		prompt += ".\n"
	}

	prompt += "\nWrite a SHORT, CONCISE description (2-3 sentences MAX) that:\n"
	prompt += "- Briefly describes the project purpose and your role\n"
	prompt += "- Mentions key technologies used\n"
	prompt += "- Includes one measurable result or impact\n"
	prompt += "- Uses action verbs without first person (no 'I', 'my', 'me')\n"
	if jobContext != "" {
		prompt += "- Emphasizes technologies and skills relevant to the target position\n"
	}
	prompt += "\nCRITICAL LANGUAGE REQUIREMENT:\n"
	prompt += "Carefully analyze the language used in the candidate's CV information above.\n"
	prompt += "If the CV contains Vietnamese text (e.g., 'Đại học', 'Công ty', 'Kinh nghiệm', Vietnamese names/addresses), write your response in Vietnamese.\n"
	prompt += "If the CV is in English, write your response in English.\n"
	prompt += "Match the language of the CV exactly - do not translate or mix languages.\n"
	prompt += "\nProvide only the paragraph text without any headers, labels, or additional explanation."

	return prompt
}

// callChatGPT makes an API call to OpenAI's ChatGPT
func (uc *ResumeUseCase) callChatGPT(ctx context.Context, prompt string) (string, error) {
	if uc.openAIKey == "" {
		return "", errors.BadRequest("OPENAI_API_KEY_NOT_CONFIGURED", "OpenAI API key is not configured")
	}

	uc.log.Info("Calling ChatGPT API with prompt length: ", len(prompt))

	// Create OpenAI client
	client := openai.NewClient(uc.openAIKey)

	// Call ChatGPT
	response, err := client.CreateChatCompletion(ctx, prompt)
	if err != nil {
		uc.log.Errorf("Failed to call ChatGPT: %v", err)
		return "", errors.InternalServer("CHATGPT_API_ERROR", "Failed to generate CV description")
	}

	return response, nil
}

// EvaluateAndSaveResume evaluates a resume using parser service and saves the result
func (uc *ResumeUseCase) EvaluateAndSaveResume(ctx context.Context, resumeID, userID string) (*Resume, error) {
	// Get the resume
	resume, err := uc.repo.GetResume(ctx, resumeID)
	if err != nil {
		return nil, err
	}

	// Check if resume exists
	if resume == nil {
		return nil, ErrResumeNotFound
	}

	// Verify user owns this resume
	if resume.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Check if parser client is available
	if uc.parserClient == nil {
		return nil, errors.InternalServer("PARSER_CLIENT_NOT_AVAILABLE", "Parser client is not initialized")
	}

	// Call evaluate service
	evaluateResp, err := uc.parserClient.EvaluateResume(ctx, userID, resume.ResumeDetail)
	if err != nil {
		uc.log.Errorf("Failed to evaluate resume: %v", err)
		return nil, errors.InternalServer("EVALUATE_FAILED", "Failed to evaluate resume")
	}

	// Check if evaluation was successful
	if !evaluateResp.Success {
		errMsg := "Evaluation failed"
		if evaluateResp.Error != "" {
			errMsg = evaluateResp.Error
		}
		return nil, errors.InternalServer("EVALUATE_FAILED", errMsg)
	}

	// Convert EvaluateResponse to ResumeEvaluation
	evaluation := &ResumeEvaluation{
		CVName:          evaluateResp.CVName,
		OverallScore:    evaluateResp.OverallScore,
		Grade:           evaluateResp.Grade,
		Strengths:       evaluateResp.Strengths,
		Weaknesses:      evaluateResp.Weaknesses,
		Recommendations: evaluateResp.Recommendations,
		JobsAnalyzed:    evaluateResp.JobsAnalyzed,
		EvaluatedAt:     time.Now(),
		Type:            "auto", // Auto evaluation based on interaction history
		JobID:           "",
		JobTitle:        "",
	}

	// Convert ScoreBreakdown
	if evaluateResp.ScoreBreakdown != nil {
		evaluation.ScoreBreakdown = &ResumeScoreBreakdown{
			SkillsScore:       evaluateResp.ScoreBreakdown.SkillsScore,
			ExperienceScore:   evaluateResp.ScoreBreakdown.ExperienceScore,
			EducationScore:    evaluateResp.ScoreBreakdown.EducationScore,
			CompletenessScore: evaluateResp.ScoreBreakdown.CompletenessScore,
			JobAlignmentScore: evaluateResp.ScoreBreakdown.JobAlignmentScore,
			PresentationScore: evaluateResp.ScoreBreakdown.PresentationScore,
		}
	}

	// Convert CVEdits
	if len(evaluateResp.CVEdits) > 0 {
		evaluation.CVEdits = make([]*ResumeCVEdit, 0, len(evaluateResp.CVEdits))
		for i, edit := range evaluateResp.CVEdits {
			// Generate unique ID for each edit
			editID := fmt.Sprintf("%s-%d-%d", resumeID, time.Now().Unix(), i)
			evaluation.CVEdits = append(evaluation.CVEdits, &ResumeCVEdit{
				ID:             editID,
				FieldPath:      edit.FieldPath,
				Action:         edit.Action,
				CurrentValue:   edit.CurrentValue,
				SuggestedValue: edit.SuggestedValue,
				Reason:         edit.Reason,
				Priority:       edit.Priority,
				ImpactScore:    edit.ImpactScore,
				Status:         "", // Empty means new/pending
			})
		}
	}

	// Initialize Evaluations array if nil
	if resume.ResumeDetail.Evaluations == nil {
		resume.ResumeDetail.Evaluations = make([]*ResumeEvaluation, 0)
	}

	// Append new evaluation to the array
	resume.ResumeDetail.Evaluations = append(resume.ResumeDetail.Evaluations, evaluation)

	// Update resume in database
	updatedResume, err := uc.repo.UpdateResume(ctx, resume)
	if err != nil {
		uc.log.Errorf("Failed to update resume with evaluation: %v", err)
		return nil, errors.InternalServer("UPDATE_FAILED", "Failed to save evaluation result")
	}

	uc.log.Infof("Successfully evaluated and saved resume %s with score: %.2f", resumeID, evaluation.OverallScore)
	return updatedResume, nil
}

// UpdateCVEditStatus updates the status of a specific CV edit
func (uc *ResumeUseCase) UpdateCVEditStatus(ctx context.Context, resumeID, editID, status, userID string) error {
	// Validate status
	if status != "accepted" && status != "rejected" {
		return errors.BadRequest("INVALID_STATUS", "Status must be 'accepted' or 'rejected'")
	}

	// Get resume
	resume, err := uc.repo.GetResume(ctx, resumeID)
	if err != nil {
		return err
	}

	if resume == nil {
		return ErrResumeNotFound
	}

	// Verify ownership
	if resume.UserID != userID {
		return ErrUnauthorized
	}

	// Find and update the CV edit
	found := false
	if resume.ResumeDetail != nil && resume.ResumeDetail.Evaluations != nil {
		for _, evaluation := range resume.ResumeDetail.Evaluations {
			if evaluation.CVEdits != nil {
				for _, edit := range evaluation.CVEdits {
					if edit.ID == editID {
						edit.Status = status
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
	}

	if !found {
		return errors.NotFound("CV_EDIT_NOT_FOUND", "CV edit not found")
	}

	// Update resume
	_, err = uc.repo.UpdateResume(ctx, resume)
	if err != nil {
		uc.log.Errorf("Failed to update CV edit status: %v", err)
		return errors.InternalServer("UPDATE_FAILED", "Failed to update CV edit status")
	}

	uc.log.Infof("Updated CV edit status: resume=%s, edit=%s, status=%s", resumeID, editID, status)
	return nil
}

// JobForEvaluation represents a job for evaluation selection
type JobForEvaluation struct {
	ID          string
	Title       string
	CompanyName string
	Location    string
	TimeOnSight int32 // Only for viewed jobs, 0 for saved jobs
}

// GetJobsForEvaluation returns viewed jobs (from tracking) and saved jobs for evaluation selection
func (uc *ResumeUseCase) GetJobsForEvaluation(ctx context.Context, userID string) ([]*JobForEvaluation, []*JobForEvaluation, error) {
	userIDObject, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, nil, errors.BadRequest("INVALID_USER_ID", "Invalid user ID")
	}

	// Get top 10 viewed jobs from tracking
	viewedJobs := make([]*JobForEvaluation, 0)
	topViewedJobs, err := uc.trackingUC.UserTrackingRepo.GetTopViewedJobsInLastWeek(ctx, userIDObject, 10)
	if err == nil && len(topViewedJobs) > 0 {
		for _, tracking := range topViewedJobs {
			job, err := uc.jobRepo.GetJobPosting(ctx, tracking.JobID.Hex())
			if err != nil || job == nil {
				continue
			}
			companyName := ""
			if job.Company != nil {
				companyName = job.Company.Name
			}
			viewedJobs = append(viewedJobs, &JobForEvaluation{
				ID:          job.ID,
				Title:       job.Title,
				CompanyName: companyName,
				Location:    job.Location,
				TimeOnSight: tracking.TimeOnSight,
			})
		}
	}

	// Get saved jobs from user
	savedJobs := make([]*JobForEvaluation, 0)
	savedJobIDs, err := uc.userRepo.GetSavedJobIDs(ctx, userID)
	if err == nil && len(savedJobIDs) > 0 {
		for _, jobID := range savedJobIDs {
			job, err := uc.jobRepo.GetJobPosting(ctx, jobID)
			if err != nil || job == nil {
				continue
			}
			companyName := ""
			if job.Company != nil {
				companyName = job.Company.Name
			}
			savedJobs = append(savedJobs, &JobForEvaluation{
				ID:          job.ID,
				Title:       job.Title,
				CompanyName: companyName,
				Location:    job.Location,
				TimeOnSight: 0,
			})
		}
	}

	return viewedJobs, savedJobs, nil
}

// EvaluateWithJD evaluates a resume with a specific JD (synchronous)
func (uc *ResumeUseCase) EvaluateWithJD(ctx context.Context, resumeID, jobID, userID string) (*Resume, *ResumeEvaluation, error) {
	// Get the resume
	resume, err := uc.repo.GetResume(ctx, resumeID)
	if err != nil {
		return nil, nil, err
	}

	// Check if resume exists
	if resume == nil {
		return nil, nil, ErrResumeNotFound
	}

	// Verify user owns this resume
	if resume.UserID != userID {
		return nil, nil, ErrUnauthorized
	}

	// Get the job posting
	job, err := uc.jobRepo.GetJobPosting(ctx, jobID)
	if err != nil {
		return nil, nil, errors.InternalServer("JOB_FETCH_ERROR", "Failed to get job posting")
	}
	if job == nil {
		return nil, nil, errors.NotFound("JOB_NOT_FOUND", "Job posting not found")
	}

	// Check if parser client is available
	if uc.parserClient == nil {
		return nil, nil, errors.InternalServer("PARSER_CLIENT_NOT_AVAILABLE", "Parser client is not initialized")
	}

	// Call evaluate service with specific JD
	evaluateResp, err := uc.parserClient.EvaluateResumeWithJD(ctx, resume.ResumeDetail, job)
	if err != nil {
		uc.log.Errorf("Failed to evaluate resume with JD: %v", err)
		return nil, nil, errors.InternalServer("EVALUATE_FAILED", "Failed to evaluate resume")
	}

	// Check if evaluation was successful
	if !evaluateResp.Success {
		errMsg := "Evaluation failed"
		if evaluateResp.Error != "" {
			errMsg = evaluateResp.Error
		}
		return nil, nil, errors.InternalServer("EVALUATE_FAILED", errMsg)
	}

	// Get company name
	companyName := ""
	if job.Company != nil {
		companyName = job.Company.Name
	}

	// Convert EvaluateResponse to ResumeEvaluation
	evaluation := &ResumeEvaluation{
		CVName:          evaluateResp.CVName,
		OverallScore:    evaluateResp.OverallScore,
		Grade:           evaluateResp.Grade,
		Strengths:       evaluateResp.Strengths,
		Weaknesses:      evaluateResp.Weaknesses,
		Recommendations: evaluateResp.Recommendations,
		JobsAnalyzed:    1,
		EvaluatedAt:     time.Now(),
		Type:            "manual",
		JobID:           jobID,
		JobTitle:        job.Title + " - " + companyName,
	}

	// Convert ScoreBreakdown
	if evaluateResp.ScoreBreakdown != nil {
		evaluation.ScoreBreakdown = &ResumeScoreBreakdown{
			SkillsScore:       evaluateResp.ScoreBreakdown.SkillsScore,
			ExperienceScore:   evaluateResp.ScoreBreakdown.ExperienceScore,
			EducationScore:    evaluateResp.ScoreBreakdown.EducationScore,
			CompletenessScore: evaluateResp.ScoreBreakdown.CompletenessScore,
			JobAlignmentScore: evaluateResp.ScoreBreakdown.JobAlignmentScore,
			PresentationScore: evaluateResp.ScoreBreakdown.PresentationScore,
		}
	}

	// Convert CVEdits
	if len(evaluateResp.CVEdits) > 0 {
		evaluation.CVEdits = make([]*ResumeCVEdit, 0, len(evaluateResp.CVEdits))
		for i, edit := range evaluateResp.CVEdits {
			// Generate unique ID for each edit
			editID := fmt.Sprintf("%s-manual-%d-%d", resumeID, time.Now().Unix(), i)
			evaluation.CVEdits = append(evaluation.CVEdits, &ResumeCVEdit{
				ID:             editID,
				FieldPath:      edit.FieldPath,
				Action:         edit.Action,
				CurrentValue:   edit.CurrentValue,
				SuggestedValue: edit.SuggestedValue,
				Reason:         edit.Reason,
				Priority:       edit.Priority,
				ImpactScore:    edit.ImpactScore,
				Status:         "", // Empty means new/pending
			})
		}
	}

	// Initialize Evaluations array if nil
	if resume.ResumeDetail.Evaluations == nil {
		resume.ResumeDetail.Evaluations = make([]*ResumeEvaluation, 0)
	}

	// Append new evaluation to the array
	resume.ResumeDetail.Evaluations = append(resume.ResumeDetail.Evaluations, evaluation)

	// Update resume in database
	updatedResume, err := uc.repo.UpdateResume(ctx, resume)
	if err != nil {
		uc.log.Errorf("Failed to update resume with evaluation: %v", err)
		return nil, nil, errors.InternalServer("UPDATE_FAILED", "Failed to save evaluation result")
	}

	uc.log.Infof("Successfully evaluated resume %s with JD %s, score: %.2f", resumeID, jobID, evaluation.OverallScore)
	return updatedResume, evaluation, nil
}

func (uc *ResumeUseCase) ScoreWithJD(ctx context.Context, resumeID, jobID, userID string) (*ScoreWithJD, error) {
	// Get the resume
	resume, err := uc.repo.GetResume(ctx, resumeID)
	if err != nil {
		return nil, err
	}

	// Check if resume exists
	if resume == nil {
		return nil, ErrResumeNotFound
	}

	// Verify user owns this resume
	if resume.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Get the job posting
	job, err := uc.jobRepo.GetJobPosting(ctx, jobID)
	if err != nil {
		return nil, errors.InternalServer("JOB_FETCH_ERROR", "Failed to get job posting")
	}
	if job == nil {
		return nil, errors.NotFound("JOB_NOT_FOUND", "Job posting not found")
	}

	// Check if parser client is available
	if uc.parserClient == nil {
		return nil, errors.InternalServer("PARSER_CLIENT_NOT_AVAILABLE", "Parser client is not initialized")
	}

	// Call evaluate service with specific JD
	evaluateResp, err := uc.parserClient.EvaluateResumeWithJD(ctx, resume.ResumeDetail, job)
	if err != nil {
		uc.log.Errorf("Failed to evaluate resume with JD: %v", err)
		return nil, errors.InternalServer("EVALUATE_FAILED", "Failed to evaluate resume")
	}

	// Check if evaluation was successful
	if !evaluateResp.Success {
		errMsg := "Evaluation failed"
		if evaluateResp.Error != "" {
			errMsg = evaluateResp.Error
		}
		return nil, errors.InternalServer("EVALUATE_FAILED", errMsg)
	}

	// Get company name
	companyName := ""
	if job.Company != nil {
		companyName = job.Company.Name
	}

	// Convert EvaluateResponse to ResumeEvaluation
	evaluation := &ResumeEvaluation{
		CVName:          evaluateResp.CVName,
		OverallScore:    evaluateResp.OverallScore,
		Grade:           evaluateResp.Grade,
		Strengths:       evaluateResp.Strengths,
		Weaknesses:      evaluateResp.Weaknesses,
		Recommendations: evaluateResp.Recommendations,
		JobsAnalyzed:    1,
		EvaluatedAt:     time.Now(),
		Type:            "manual",
		JobID:           jobID,
		JobTitle:        job.Title + " - " + companyName,
	}

	// Convert ScoreBreakdown
	if evaluateResp.ScoreBreakdown != nil {
		evaluation.ScoreBreakdown = &ResumeScoreBreakdown{
			SkillsScore:       evaluateResp.ScoreBreakdown.SkillsScore,
			ExperienceScore:   evaluateResp.ScoreBreakdown.ExperienceScore,
			EducationScore:    evaluateResp.ScoreBreakdown.EducationScore,
			CompletenessScore: evaluateResp.ScoreBreakdown.CompletenessScore,
			JobAlignmentScore: evaluateResp.ScoreBreakdown.JobAlignmentScore,
			PresentationScore: evaluateResp.ScoreBreakdown.PresentationScore,
		}
	}
	skillMissing := make([]string, 0)
	// Convert CVEdits
	if len(evaluateResp.CVEdits) > 0 {
		for _, edit := range evaluateResp.CVEdits {
			if edit.FieldPath == "skills" {
				skillMissing = append(skillMissing, edit.SuggestedValue)
			}
		}
	}

	score := &ScoreWithJD{
		JobID:          jobID,
		OverallScore:   evaluation.OverallScore,
		ScoreBreakdown: evaluation.ScoreBreakdown,
		MissingSkill:   skillMissing,
	}

	uc.log.Infof("Successfully score resume %s with JD %s, score: %.2f", resumeID, jobID, evaluation.OverallScore)
	return score, nil
}

// Error definitions
var (
	ErrResumeNotFound      = errors.NotFound("RESUME_NOT_FOUND", "Resume not found")
	ErrInvalidResume       = errors.BadRequest("INVALID_RESUME", "Invalid resume data")
	ErrUnauthorized        = errors.Forbidden("UNAUTHORIZED", "You don't have permission to access this resume")
	ErrResumeAlreadyExists = errors.BadRequest("RESUME_ALREADY_EXISTS", "You already have a resume. Please update it instead of creating a new one")
)
