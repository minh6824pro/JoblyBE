package server

import (
	"JobblyBE/internal/data"
	"JobblyBE/pkg/kafkax"
	"JobblyBE/pkg/middleware/auth"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/imroc/req/v3"
)

const (
	MaxUploadSize = 10 << 20 // 10 MB
)

type ServiceErrorResult struct {
	Detail string `json:"detail"`
}

// ParserResponse represents the response from parser service
type ParserResponse struct {
	Success bool   `json:"success"`
	CVData  CVData `json:"cv_data"`
	Message string `json:"message"`
}

type CVData struct {
	Name           string       `json:"name"`
	Email          string       `json:"email"`
	Phone          string       `json:"phone"`
	Summary        string       `json:"summary"`
	Skills         []string     `json:"skills"`
	Education      []Education  `json:"education"`
	Experience     []Experience `json:"experience"`
	Projects       []Project    `json:"projects"`
	Certifications []string     `json:"certifications"`
	Languages      []string     `json:"languages"`
	Achievements   []string     `json:"achievements"`
}

type Education struct {
	Degree         string  `json:"degree"`
	Institution    string  `json:"institution"`
	GraduationYear int     `json:"graduation_year"`
	Description    string  `json:"description"`
	GPA            float64 `json:"gpa"`
}

type Experience struct {
	Title            string   `json:"title"`
	Company          string   `json:"company"`
	Duration         string   `json:"duration"`
	Description      string   `json:"description"`
	Responsibilities []string `json:"responsibilities"`
	Achievements     []string `json:"achievements"`
}

type Project struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies"`
	Url          string   `json:"url"`
	Duration     string   `json:"duration"`
	Role         string   `json:"role"`
	Achievements []string `json:"achievements"`
}

type UploadHandler struct {
	parserURL string
	jwtSecret string
	log       *log.Helper
	cli       *req.Client
	db        *mongo.Database
	producer  *kafkax.Producer
	jobRepo   *data.ResumeParseJobRepo
	topic     string
}

// UploadHandlerConfig holds configuration for UploadHandler
type UploadHandlerConfig struct {
	ParserURL      string
	JwtSecret      string
	DatabaseSource string
	DatabaseName   string
	KafkaTopic     string
}

func NewUploadHandler(
	config *UploadHandlerConfig,
	producer *kafkax.Producer,
	jobRepo *data.ResumeParseJobRepo,
	logger log.Logger,
) (*UploadHandler, error) {
	logHelper := log.NewHelper(logger)

	// Create MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.DatabaseSource))
	if err != nil {
		logHelper.Errorf("failed to connect to mongodb: %v", err)
		return nil, err
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		logHelper.Errorf("failed to ping mongodb: %v", err)
		return nil, err
	}

	logHelper.Info("successfully connected to mongodb for upload handler")

	db := client.Database(config.DatabaseName)

	return &UploadHandler{
		parserURL: config.ParserURL,
		jwtSecret: config.JwtSecret,
		log:       logHelper,
		db:        db,
		producer:  producer,
		jobRepo:   jobRepo,
		topic:     config.KafkaTopic,
		cli: req.C().
			SetBaseURL(config.ParserURL).
			SetTimeout(5 * time.Minute),
	}, nil
}

// sendToParserService forwards multipart file to external parser service (for sync mode)
func (h *UploadHandler) sendToParserService(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*ParserResponse, error) {
	var result ParserResponse
	var errRS ServiceErrorResult

	// Forward the multipart file directly using SetFileReader
	resp, err := h.cli.R().
		SetContext(ctx).
		SetFileReader("file", header.Filename, file).
		SetSuccessResult(&result).
		SetErrorResult(&errRS).
		Post("/parse/pdf")

	if err != nil {
		h.log.Errorf("failed to send request to parser service: %v", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if !resp.IsSuccessState() {
		h.log.Errorf("parser service returned error: %s", errRS.Detail)
		return nil, fmt.Errorf("parser service error: %s", errRS.Detail)
	}

	if !result.Success {
		h.log.Errorf("parser service failed: %s", result.Message)
		return nil, fmt.Errorf("parser failed: %s", result.Message)
	}

	return &result, nil
}

// convertToDataResume converts ParserResponse to data.Resume
func (h *UploadHandler) convertToDataResume(parserResp *ParserResponse) *data.Resume {
	cvData := parserResp.CVData

	// Convert all education
	educationList := make([]data.Education, 0, len(cvData.Education))
	for _, edu := range cvData.Education {
		educationList = append(educationList, data.Education{
			Degree:         edu.Degree,
			Institution:    edu.Institution,
			GraduationYear: fmt.Sprintf("%d", edu.GraduationYear),
		})
	}

	// Convert all experience
	experienceList := make([]data.Experience, 0, len(cvData.Experience))
	for _, exp := range cvData.Experience {
		experienceList = append(experienceList, data.Experience{
			Title:            exp.Title,
			Company:          exp.Company,
			Duration:         exp.Duration,
			Description:      exp.Description,
			Responsibilities: exp.Responsibilities,
			Achievements:     exp.Achievements,
		})
	}

	// Convert all projects
	projectList := make([]data.Project, 0, len(cvData.Projects))
	for _, proj := range cvData.Projects {
		projectList = append(projectList, data.Project{
			Name:         proj.Name,
			Description:  proj.Description,
			Technologies: proj.Technologies,
			Url:          proj.Url,
			Duration:     proj.Duration,
			Role:         proj.Role,
			Achievements: proj.Achievements,
		})
	}

	resumeDetail := data.ResumeDetail{
		Name:           cvData.Name,
		Email:          cvData.Email,
		Phone:          cvData.Phone,
		Summary:        cvData.Summary,
		Skills:         cvData.Skills,
		Education:      educationList,
		Experience:     experienceList,
		Projects:       projectList,
		Certifications: cvData.Certifications,
		Languages:      cvData.Languages,
	}

	return &data.Resume{
		ID:           primitive.NewObjectID(),
		ResumeDetail: resumeDetail,
		Version:      1,
		CreatedAt:    time.Now(),
	}
}

// extractToken extracts JWT token from Authorization header
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Expected format: "Bearer <token>"
	const prefix = "Bearer "
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		return authHeader[len(prefix):]
	}

	return ""
}

// HandleUploadResumeAsync handles resume file upload with async processing via Kafka
func (h *UploadHandler) HandleUploadResumeAsync(w http.ResponseWriter, r *http.Request) {
	// Parse JWT from Authorization header
	token := extractToken(r)
	var userID string

	if token == "" {
		h.log.Warn("No token provided")
		http.Error(w, "Unauthorized: token required", http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateAccessToken(token, h.jwtSecret)
	if err != nil || claims == nil {
		h.log.Warnf("Invalid or expired token: %v", err)
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}
	userID = claims.UserID
	h.log.Infof("User authenticated: %s", userID)

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		h.log.Errorf("failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		h.log.Errorf("failed to get file from form: %v", err)
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	h.log.Infof("received file: %s, size: %d bytes", header.Filename, header.Size)

	// Validate file type (must be PDF)
	contentType := header.Header.Get("Content-Type")
	if contentType != "application/pdf" {
		h.log.Errorf("invalid file type: %s", contentType)
		http.Error(w, "Only PDF files are allowed", http.StatusBadRequest)
		return
	}

	// Read file data
	ctx := r.Context()
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		h.log.Errorf("failed to parse user id: %v", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Verify user exists
	var user data.User
	err = h.db.Collection(data.CollectionUser).FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		h.log.Errorf("failed to find user: %v", err)
		http.Error(w, "Failed to find user", http.StatusInternalServerError)
		return
	}

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		h.log.Errorf("failed to read file: %v", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Create parse job in database
	job := &data.ResumeParseJob{
		UserID:   userObjID,
		FileName: header.Filename,
		FileData: fileData,
		Status:   data.ResumeParseStatusPending,
	}

	createdJob, err := h.jobRepo.Create(ctx, job)
	if err != nil {
		h.log.Errorf("failed to create parse job: %v", err)
		http.Error(w, "Failed to create parse job", http.StatusInternalServerError)
		return
	}

	// Send message to Kafka
	kafkaMsg := ResumeParseMessage{
		JobID:    createdJob.ID.Hex(),
		UserID:   userID,
		FileName: header.Filename,
	}

	if err := h.producer.SendMessage(ctx, h.topic, createdJob.ID.Hex(), kafkaMsg); err != nil {
		h.log.Errorf("failed to send message to Kafka: %v", err)
		// Update job status to failed
		h.jobRepo.UpdateStatus(ctx, createdJob.ID.Hex(), data.ResumeParseStatusFailed, "Failed to queue job", "")
		http.Error(w, "Failed to queue resume parsing job", http.StatusInternalServerError)
		return
	}

	h.log.Infof("Resume parse job created and queued: %s", createdJob.ID.Hex())

	// Return success response with job ID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	response := map[string]interface{}{
		"success": true,
		"message": "Resume upload accepted. Processing will be done asynchronously.",
		"job_id":  createdJob.ID.Hex(),
		"status":  string(data.ResumeParseStatusPending),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Errorf("failed to encode response: %v", err)
	}
}

// HandleGetJobStatus returns the status of a resume parse job
func (h *UploadHandler) HandleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	// Parse JWT from Authorization header
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized: token required", http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateAccessToken(token, h.jwtSecret)
	if err != nil || claims == nil {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	// Get job ID from query
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	// Get job from database
	job, err := h.jobRepo.GetByID(r.Context(), jobID)
	if err != nil {
		h.log.Errorf("failed to get job: %v", err)
		http.Error(w, "Failed to get job", http.StatusInternalServerError)
		return
	}

	if job == nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Check if job belongs to user
	if job.UserID.Hex() != claims.UserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Return job status
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"job_id":     job.ID.Hex(),
		"status":     string(job.Status),
		"file_name":  job.FileName,
		"created_at": job.CreatedAt,
		"updated_at": job.UpdatedAt,
	}

	if job.ErrorMessage != "" {
		response["error_message"] = job.ErrorMessage
	}

	if !job.ResumeID.IsZero() {
		response["resume_id"] = job.ResumeID.Hex()
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Errorf("failed to encode response: %v", err)
	}
}

// HandleUploadResume handles resume file upload (sync mode - kept for backward compatibility)
func (h *UploadHandler) HandleUploadResume(w http.ResponseWriter, r *http.Request) {
	// Parse JWT from Authorization header manually
	token := extractToken(r)
	var userID string

	if token != "" {
		claims, err := auth.ValidateAccessToken(token, h.jwtSecret)
		if err == nil && claims != nil {
			userID = claims.UserID
			h.log.Infof("User authenticated: %s", userID)
		} else {
			h.log.Warnf("Invalid or expired token: %v", err)
		}
	} else {
		h.log.Info("No token provided, processing as anonymous")
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		h.log.Errorf("failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		h.log.Errorf("failed to get file from form: %v", err)
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	h.log.Infof("received file: %s, size: %d bytes", header.Filename, header.Size)

	// Validate file type (must be PDF)
	contentType := header.Header.Get("Content-Type")
	if contentType != "application/pdf" {
		h.log.Errorf("invalid file type: %s", contentType)
		http.Error(w, "Only PDF files are allowed", http.StatusBadRequest)
		return
	}

	// Forward multipart file directly to parser service (no reading/conversion)
	parserResp, err := h.sendToParserService(r.Context(), file, header)
	if err != nil {
		h.log.Errorf("failed to parse resume: %v", err)
		http.Error(w, "Failed to parse resume", http.StatusInternalServerError)
		return
	}

	h.log.Infof("Successfully parsed resume from parser service")

	// Convert to data.Resume
	resume := h.convertToDataResume(parserResp)

	// Save to MongoDB using data.Data's db client
	ctx := r.Context()
	collection := h.db.Collection(data.CollectionUser)
	userIDObj, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		h.log.Errorf("failed to parse user id: %v", err)
		http.Error(w, "Failed to parse user id", http.StatusBadRequest)
		return
	}

	// Verify user exists
	var user data.User
	err = collection.FindOne(ctx, bson.M{"_id": userIDObj}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			h.log.Errorf("user not found: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		h.log.Errorf("failed to find user: %v", err)
		http.Error(w, "Failed to find user", http.StatusInternalServerError)
		return
	}

	filter := bson.M{"_id": userIDObj} // tìm user theo ID
	update := bson.M{
		"$push": bson.M{
			"resume": resume, // thêm resume mới vào array
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// not create if user not exists
	opts := options.Update().SetUpsert(false)

	_, err = collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		h.log.Errorf("failed to save resume to database: %v", err)
		http.Error(w, "Failed to save resume", http.StatusInternalServerError)
		return
	}

	h.log.Infof("Resume saved to database with ID: %v", resume.ID.Hex())

	// Push message to Kafka for evaluation
	evaluateMsg := map[string]interface{}{
		"resume_id": resume.ID.Hex(),
		"user_id":   userID,
	}
	if err := h.producer.SendMessage(ctx, "evaluate", resume.ID.Hex(), evaluateMsg); err != nil {
		h.log.Errorf("Failed to push evaluate message to Kafka: %v", err)
		// Don't fail the request, just log the error
	} else {
		h.log.Infof("Pushed evaluate message to Kafka for resume: %s", resume.ID.Hex())
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Return the parsed resume data
	response := map[string]interface{}{
		"success":   true,
		"message":   "Resume uploaded, parsed and saved successfully",
		"resume_id": resume.ID.Hex(),
		"cv_data":   parserResp.CVData,
	}

	// Marshal response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Errorf("failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
