package server

import (
	"JobblyBE/internal/data"
	"JobblyBE/pkg/middleware/auth"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"JobblyBE/internal/conf"

	"github.com/imroc/req/v3"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	Certifications []string     `json:"certifications"`
	Languages      []string     `json:"languages"`
}

type Education struct {
	Degree         string  `json:"degree"`
	Institution    string  `json:"institution"`
	GraduationYear int     `json:"graduation_year"`
	GPA            float64 `json:"gpa"`
}

type Experience struct {
	Title            string   `json:"title"`
	Company          string   `json:"company"`
	Duration         string   `json:"duration"`
	Responsibilities []string `json:"responsibilities"`
	Achievements     []string `json:"achievements"`
}

type UploadHandler struct {
	parserURL string
	jwtSecret string
	log       *log.Helper
	cli       *req.Client
	db        *mongo.Database
}

func NewUploadHandler(parserURL string, logger log.Logger, databaseSource string, databaseName string, jwtSecret string) (*UploadHandler, error) {

	log := log.NewHelper(logger)

	// Create MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(databaseSource))
	if err != nil {
		log.Errorf("failed to connect to mongodb: %v", err)
		return nil, err
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		log.Errorf("failed to ping mongodb: %v", err)
		return nil, err
	}

	log.Info("successfully connected to mongodb")

	// Get database name from config or use default
	dbName := databaseName
	db := client.Database(dbName)

	return &UploadHandler{
		parserURL: parserURL,
		jwtSecret: jwtSecret,
		log:       log,
		db:        db,
		cli: req.C().
			SetBaseURL(parserURL).
			SetTimeout(5 * time.Minute), // 5 minutes timeout for parsing
	}, nil
}

// sendToParserService forwards multipart file to external parser service
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

	// Convert first education (if exists)
	var education data.Education
	if len(cvData.Education) > 0 {
		edu := cvData.Education[0]
		education = data.Education{
			Degree:         edu.Degree,
			Institution:    edu.Institution,
			GraduationYear: fmt.Sprintf("%d", edu.GraduationYear),
		}
	}

	// Convert first experience (if exists)
	var experience data.Experience
	if len(cvData.Experience) > 0 {
		exp := cvData.Experience[0]
		experience = data.Experience{
			Title:            exp.Title,
			Company:          exp.Company,
			Duration:         exp.Duration,
			Responsibilities: exp.Responsibilities,
			Achievements:     exp.Achievements,
		}
	}

	resumeDetail := data.ResumeDetail{
		Name:           cvData.Name,
		Email:          cvData.Email,
		Phone:          cvData.Phone,
		Summary:        cvData.Summary,
		Skills:         cvData.Skills,
		Education:      education,
		Experience:     experience,
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

// HandleUploadResume handles resume file upload and sends to parser service
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

	// Check if user already has a resume
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

	if len(user.Resume) > 0 {
		h.log.Warnf("user already has a resume")
		http.Error(w, "You already have a resume. Please update it instead of creating a new one", http.StatusBadRequest)
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

type Database struct {
	db  *mongo.Database
	log *log.Helper
}

func NewDatabase(c *conf.Data, logger log.Logger) (*Database, error) {
	helper := log.NewHelper(logger)

	// Create MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(c.Database.Source))
	if err != nil {
		helper.Errorf("failed to connect to mongodb: %v", err)
		return nil, err
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		helper.Errorf("failed to ping mongodb: %v", err)
		return nil, err
	}

	helper.Info("successfully connected to mongodb")

	// Get database name from config or use default
	dbName := c.Database.Name
	db := client.Database(dbName)

	return &Database{
		db:  db,
		log: helper,
	}, nil
}
