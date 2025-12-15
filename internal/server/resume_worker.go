package server

import (
	"JobblyBE/internal/data"
	"JobblyBE/pkg/kafkax"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ResumeParseMessage represents the Kafka message for resume parsing
type ResumeParseMessage struct {
	JobID    string `json:"job_id"`
	UserID   string `json:"user_id"`
	FileName string `json:"file_name"`
}

// ResumeWorker handles resume parsing jobs from Kafka
type ResumeWorker struct {
	consumer   *kafkax.Consumer
	producer   *kafkax.Producer
	hub        *Hub
	jobRepo    *data.ResumeParseJobRepo
	parserURL  string
	db         *mongo.Database
	log        *log.Helper
	httpClient *http.Client
}

// ResumeWorkerConfig holds configuration for the resume worker
type ResumeWorkerConfig struct {
	Brokers       []string
	Username      string
	Password      string
	EnableSASL    bool
	Timeout       time.Duration
	Topic         string
	GroupID       string
	NumPartitions int
	NumWorkers    int
	ParserURL     string
}

// NewResumeWorker creates a new resume worker
func NewResumeWorker(
	config *ResumeWorkerConfig,
	producer *kafkax.Producer,
	hub *Hub,
	jobRepo *data.ResumeParseJobRepo,
	db *mongo.Database,
	logger log.Logger,
) *ResumeWorker {
	consumerConfig := &kafkax.ConsumerConfig{
		Brokers:       config.Brokers,
		Username:      config.Username,
		Password:      config.Password,
		EnableSASL:    config.EnableSASL,
		Timeout:       config.Timeout,
		Topic:         config.Topic,
		GroupID:       config.GroupID,
		NumPartitions: config.NumPartitions,
		NumWorkers:    config.NumWorkers,
	}

	consumer := kafkax.NewConsumer(consumerConfig, logger)

	return &ResumeWorker{
		consumer:   consumer,
		producer:   producer,
		hub:        hub,
		jobRepo:    jobRepo,
		parserURL:  config.ParserURL,
		db:         db,
		log:        log.NewHelper(logger),
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// Start starts the resume worker
func (w *ResumeWorker) Start() {
	w.log.Info("Starting resume parsing worker...")
	w.consumer.Start(w.handleMessage)
}

// Stop stops the resume worker
func (w *ResumeWorker) Stop() error {
	w.log.Info("Stopping resume parsing worker...")
	return w.consumer.Stop()
}

// handleMessage processes a Kafka message for resume parsing
func (w *ResumeWorker) handleMessage(ctx context.Context, msg kafka.Message) error {
	var parseMsg ResumeParseMessage
	if err := json.Unmarshal(msg.Value, &parseMsg); err != nil {
		w.log.Errorf("Failed to unmarshal message: %v", err)
		return err
	}

	w.log.Infof("Processing resume parse job: %s for user: %s", parseMsg.JobID, parseMsg.UserID)

	// Update status to processing
	if err := w.jobRepo.UpdateStatus(ctx, parseMsg.JobID, data.ResumeParseStatusProcessing, "", ""); err != nil {
		w.log.Errorf("Failed to update job status to processing: %v", err)
	}

	// Notify user via WebSocket
	w.hub.SendJobStatus(parseMsg.UserID, &JobStatusPayload{
		JobID:  parseMsg.JobID,
		Status: string(data.ResumeParseStatusProcessing),
	})

	// Get job from database to retrieve file data
	job, err := w.jobRepo.GetByID(ctx, parseMsg.JobID)
	if err != nil || job == nil {
		w.log.Errorf("Failed to get job: %v", err)
		w.handleJobFailure(ctx, parseMsg.JobID, parseMsg.UserID, "Job not found")
		return err
	}

	// Parse resume
	parserResp, err := w.sendToParserService(ctx, job.FileData, job.FileName)
	if err != nil {
		w.log.Errorf("Failed to parse resume: %v", err)
		w.handleJobFailure(ctx, parseMsg.JobID, parseMsg.UserID, err.Error())
		return err
	}

	// Convert to resume and save
	resume := w.convertToDataResume(parserResp)

	// Save to user's resume array
	resumeID, err := w.saveResumeToUser(ctx, parseMsg.UserID, resume)
	if err != nil {
		w.log.Errorf("Failed to save resume: %v", err)
		w.handleJobFailure(ctx, parseMsg.JobID, parseMsg.UserID, err.Error())
		return err
	}

	// Update job status to completed
	if err := w.jobRepo.UpdateStatus(ctx, parseMsg.JobID, data.ResumeParseStatusCompleted, "", resumeID); err != nil {
		w.log.Errorf("Failed to update job status to completed: %v", err)
	}

	// Clear file data to save space
	if err := w.jobRepo.ClearFileData(ctx, parseMsg.JobID); err != nil {
		w.log.Errorf("Failed to clear file data: %v", err)
	}

	// Push message to Kafka for evaluation
	evaluateMsg := map[string]interface{}{
		"resume_id": resumeID,
		"user_id":   parseMsg.UserID,
	}
	if err := w.producer.SendMessage(ctx, "evaluate", resumeID, evaluateMsg); err != nil {
		w.log.Errorf("Failed to push evaluate message to Kafka: %v", err)
		// Don't fail the job, just log the error
	} else {
		w.log.Infof("Pushed evaluate message to Kafka for resume: %s", resumeID)
	}

	// Log CVData for debugging
	w.log.Infof("Sending CVData via WebSocket - Education count: %d, Experience count: %d",
		len(parserResp.CVData.Education), len(parserResp.CVData.Experience))

	// Notify user via WebSocket with complete data
	w.hub.SendJobStatus(parseMsg.UserID, &JobStatusPayload{
		JobID:    parseMsg.JobID,
		Status:   string(data.ResumeParseStatusCompleted),
		ResumeID: resumeID,
		CVData:   parserResp.CVData,
	})

	w.log.Infof("Resume parse job completed: %s, resume ID: %s", parseMsg.JobID, resumeID)
	return nil
}

// handleJobFailure handles job failure
func (w *ResumeWorker) handleJobFailure(ctx context.Context, jobID, userID, errorMsg string) {
	if err := w.jobRepo.UpdateStatus(ctx, jobID, data.ResumeParseStatusFailed, errorMsg, ""); err != nil {
		w.log.Errorf("Failed to update job status to failed: %v", err)
	}

	// Clear file data even on failure
	if err := w.jobRepo.ClearFileData(ctx, jobID); err != nil {
		w.log.Errorf("Failed to clear file data: %v", err)
	}

	w.hub.SendJobStatus(userID, &JobStatusPayload{
		JobID:        jobID,
		Status:       string(data.ResumeParseStatusFailed),
		ErrorMessage: errorMsg,
	})
}

// sendToParserService sends file to parser service
func (w *ResumeWorker) sendToParserService(ctx context.Context, fileData []byte, fileName string) (*ParserResponse, error) {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(fileData)); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", w.parserURL+"/parse/pdf", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("parser service error: %s", string(bodyBytes))
	}

	// Parse response
	var result ParserResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("parser failed: %s", result.Message)
	}

	return &result, nil
}

// convertToDataResume converts ParserResponse to data.Resume
func (w *ResumeWorker) convertToDataResume(parserResp *ParserResponse) *data.Resume {
	cvData := parserResp.CVData

	// Convert all education
	educationList := make([]data.Education, 0, len(cvData.Education))
	for _, edu := range cvData.Education {
		educationList = append(educationList, data.Education{
			Degree:         edu.Degree,
			Institution:    edu.Institution,
			GraduationYear: fmt.Sprintf("%d", edu.GraduationYear),
			Description:    edu.Description,
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
		Achievements:   cvData.Achievements,
	}

	return &data.Resume{
		ID:           primitive.NewObjectID(),
		ResumeDetail: resumeDetail,
		Version:      1,
		CreatedAt:    time.Now(),
	}
}

// saveResumeToUser saves resume to user's resume array
func (w *ResumeWorker) saveResumeToUser(ctx context.Context, userID string, resume *data.Resume) (string, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	// Verify user exists
	var user data.User
	err = w.db.Collection(data.CollectionUser).FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("user not found")
		}
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	filter := bson.M{"_id": userObjID}
	update := bson.M{
		"$push": bson.M{
			"resume": resume,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(false)
	_, err = w.db.Collection(data.CollectionUser).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return "", fmt.Errorf("failed to save resume: %w", err)
	}

	return resume.ID.Hex(), nil
}
