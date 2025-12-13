package biz

import (
	"JobblyBE/pkg/configx"
	"JobblyBE/pkg/openai"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
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
	Certifications []string
	Languages      []string
}

type Education struct {
	Degree         string
	Institution    string
	GraduationYear string
}

type Experience struct {
	Title            string
	Company          string
	Duration         string
	Responsibilities []string
	Achievements     []string
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
	repo       ResumeRepo
	trackingUC *UserTrackingUseCase
	openAIKey  string
	log        *log.Helper
}

// NewResumeUseCase creates a new resume use case
func NewResumeUseCase(repo ResumeRepo, trackingUC *UserTrackingUseCase, logger log.Logger) *ResumeUseCase {
	return &ResumeUseCase{
		repo:       repo,
		trackingUC: trackingUC,
		openAIKey:  configx.GetEnvOrString("OPENAI_API_KEY", ""),
		log:        log.NewHelper(logger),
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

	return uc.repo.CreateResume(ctx, resume)
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

	return uc.repo.UpdateResume(ctx, resume)
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

// GenerateCVDescription generates a CV description using ChatGPT based on CV data and most viewed job
func (uc *ResumeUseCase) GenerateCVDescription(ctx context.Context, resumeID, userID string) (string, error) {
	// Get resume
	resume, err := uc.GetResume(ctx, resumeID, userID)
	if err != nil {
		return "", err
	}

	var prompt string

	// Try to get most viewed job
	job, err := uc.trackingUC.GetMostViewedJobByUser(ctx, userID)
	if err != nil {
		// No tracking found, just use CV info
		uc.log.Info("No job tracking found for user, generating description based on CV only")
		prompt = uc.buildPromptWithCVOnly(resume.ResumeDetail)
	} else {
		// Build prompt with CV and job info
		prompt = uc.buildPromptWithJobAndCV(resume.ResumeDetail, job)
	}

	// Call ChatGPT API
	description, err := uc.callChatGPT(ctx, prompt)
	if err != nil {
		return "", err
	}

	return description, nil
}

// buildPromptWithCVOnly creates a prompt using only CV information
func (uc *ResumeUseCase) buildPromptWithCVOnly(cv *ResumeDetail) string {
	prompt := "Based on the following CV information, write a professional and compelling summary/description for this candidate:\n\n"
	prompt += "Name: " + cv.Name + "\n"
	prompt += "Email: " + cv.Email + "\n"
	if cv.Phone != "" {
		prompt += "Phone: " + cv.Phone + "\n"
	}
	if cv.Summary != "" {
		prompt += "Current Summary: " + cv.Summary + "\n"
	}

	if len(cv.Skills) > 0 {
		prompt += "\nSkills:\n"
		for _, skill := range cv.Skills {
			prompt += "- " + skill + "\n"
		}
	}

	if len(cv.Education) > 0 {
		prompt += "\nEducation:\n"
		for _, edu := range cv.Education {
			prompt += "- " + edu.Degree + " at " + edu.Institution + " (" + edu.GraduationYear + ")\n"
		}
	}

	if len(cv.Experience) > 0 {
		prompt += "\nExperience:\n"
		for _, exp := range cv.Experience {
			prompt += "- " + exp.Title + " at " + exp.Company + " (" + exp.Duration + ")\n"
			for _, resp := range exp.Responsibilities {
				prompt += "  * " + resp + "\n"
			}
		}
	}

	prompt += "\nIMPORTANT: Write the summary in FIRST PERSON perspective (using 'I', 'my', 'me'), as if the candidate is writing about themselves.\n"
	prompt += "Write a concise, professional summary (2-3 paragraphs) that highlights the key strengths, experiences, and qualifications. Focus on career achievements and what makes this candidate stand out.\n"
	prompt += "Start with statements like 'I am...', 'I have experience in...', 'My expertise includes...', etc."

	return prompt
}

// buildPromptWithJobAndCV creates a prompt using both CV and job information
func (uc *ResumeUseCase) buildPromptWithJobAndCV(cv *ResumeDetail, job *JobPosting) string {
	prompt := "Based on the following CV information and the job position the candidate is most interested in, write a professional and compelling summary/description tailored for this specific role:\n\n"
	prompt += "TARGET JOB:\n"
	prompt += "Position: " + job.Title + "\n"
	if job.Company != nil {
		prompt += "Company: " + job.Company.Name + "\n"
	}
	if job.Description != "" {
		prompt += "Job Description: " + job.Description + "\n"
	}
	if job.Requirements != "" {
		prompt += "Requirements: " + job.Requirements + "\n"
	}

	prompt += "\nCANDIDATE CV:\n"
	prompt += "Name: " + cv.Name + "\n"
	if cv.Summary != "" {
		prompt += "Current Summary: " + cv.Summary + "\n"
	}

	if len(cv.Skills) > 0 {
		prompt += "\nSkills:\n"
		for _, skill := range cv.Skills {
			prompt += "- " + skill + "\n"
		}
	}

	if len(cv.Education) > 0 {
		prompt += "\nEducation:\n"
		for _, edu := range cv.Education {
			prompt += "- " + edu.Degree + " at " + edu.Institution + " (" + edu.GraduationYear + ")\n"
		}
	}

	if len(cv.Experience) > 0 {
		prompt += "\nExperience:\n"
		for _, exp := range cv.Experience {
			prompt += "- " + exp.Title + " at " + exp.Company + " (" + exp.Duration + ")\n"
			for _, resp := range exp.Responsibilities {
				prompt += "  * " + resp + "\n"
			}
		}
	}

	prompt += "\nIMPORTANT: Write the summary in FIRST PERSON perspective (using 'I', 'my', 'me'), as if the candidate is writing about themselves.\n"
	prompt += "Write a concise, professional summary (2-3 paragraphs) that:\n"
	prompt += "1. Highlights how my experience and skills align with the target job requirements\n"
	prompt += "2. Emphasizes my relevant achievements and qualifications for this specific position\n"
	companyInfo := "this company"
	if job.Company != nil {
		companyInfo = "this role at " + job.Company.Name
	}
	prompt += "3. Shows why I would be a great fit for " + companyInfo + "\n"
	prompt += "Start with statements like 'I am...', 'I have...', 'My background includes...', etc."

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

// Error definitions
var (
	ErrResumeNotFound      = errors.NotFound("RESUME_NOT_FOUND", "Resume not found")
	ErrInvalidResume       = errors.BadRequest("INVALID_RESUME", "Invalid resume data")
	ErrUnauthorized        = errors.Forbidden("UNAUTHORIZED", "You don't have permission to access this resume")
	ErrResumeAlreadyExists = errors.BadRequest("RESUME_ALREADY_EXISTS", "You already have a resume. Please update it instead of creating a new one")
)
