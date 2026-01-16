package biz

import (
	"JobblyBE/pkg/milvusx"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// JobEmbeddingResult represents the result of embedding a job posting
type JobEmbeddingResult struct {
	JobID      string
	Embeddings []*JobEmbeddingItem
}

// JobEmbeddingItem represents a single embedding for a job
type JobEmbeddingItem struct {
	ID            string
	EmbeddingType string
	Embedding     []float32
	TextContent   string
}

// JobEmbeddingRepo interface for job embedding data layer
type JobEmbeddingRepo interface {
	// SaveJobEmbedding saves a single job embedding
	SaveJobEmbedding(ctx context.Context, jobID string, embeddingType string, embedding []float32, textContent string) error

	// SaveJobEmbeddings saves multiple job embeddings
	SaveJobEmbeddings(ctx context.Context, embeddings []*milvusx.JobEmbedding) error

	// DeleteJobEmbeddings deletes all embeddings for a job
	DeleteJobEmbeddings(ctx context.Context, jobID string) error

	// SearchSimilarJobs searches for similar jobs based on query vector
	SearchSimilarJobs(ctx context.Context, queryVector []float32, embeddingType string, topK int) ([]*SimilarJobResult, error)

	// GetJobEmbeddings retrieves embeddings for a specific job by embedding types
	GetJobEmbeddings(ctx context.Context, jobID string, embeddingTypes []string) (map[string][]float32, error)
}

// SimilarJobResult represents a similar job search result
type SimilarJobResult struct {
	JobID       string
	TextContent string
	Score       float32
}

// EmbeddingClient interface for generating embeddings
type EmbeddingClient interface {
	CreateEmbedding(ctx context.Context, text string) ([]float32, error)
	CreateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// JobEmbeddingUseCase handles job embedding business logic
type JobEmbeddingUseCase struct {
	embeddingRepo   JobEmbeddingRepo
	embeddingClient EmbeddingClient
	jobRepo         JobPostingRepo
	log             *log.Helper
}

// NewJobEmbeddingUseCase creates a new job embedding use case
func NewJobEmbeddingUseCase(
	embeddingRepo JobEmbeddingRepo,
	embeddingClient EmbeddingClient,
	jobRepo JobPostingRepo,
	logger log.Logger,
) *JobEmbeddingUseCase {
	return &JobEmbeddingUseCase{
		embeddingRepo:   embeddingRepo,
		embeddingClient: embeddingClient,
		jobRepo:         jobRepo,
		log:             log.NewHelper(logger),
	}
}

// EmbedJobPosting embeds a job posting and saves to vector database
// This method creates multiple embeddings for different aspects of the job:
// 1. Full content embedding - complete job description
// 2. Title embedding - job title only
// 3. Requirements embedding - job requirements
// 4. Skills embedding - tech stack and skills
func (uc *JobEmbeddingUseCase) EmbedJobPosting(ctx context.Context, job *JobPosting) (*JobEmbeddingResult, error) {
	uc.log.WithContext(ctx).Infof("EmbedJobPosting: %s (ID: %s)", job.Title, job.ID)

	// Prepare texts for embedding
	embeddingTexts := uc.prepareEmbeddingTexts(job)

	// Create embeddings for all texts
	var allEmbeddings []*milvusx.JobEmbedding
	now := time.Now().Unix()

	for embType, text := range embeddingTexts {
		if text == "" {
			continue
		}

		// Truncate text if too long (OpenAI has token limits)
		truncatedText := truncateText(text, 8000)

		// Generate embedding
		embedding, err := uc.embeddingClient.CreateEmbedding(ctx, truncatedText)
		if err != nil {
			uc.log.Errorf("failed to create embedding for %s: %v", embType, err)
			return nil, fmt.Errorf("failed to create embedding for %s: %w", embType, err)
		}

		// Create embedding record
		embRecord := &milvusx.JobEmbedding{
			ID:            generateEmbeddingID(job.ID, embType),
			JobID:         job.ID,
			EmbeddingType: milvusx.EmbeddingType(embType),
			Embedding:     embedding,
			TextContent:   truncatedText,
			CreatedAt:     now,
		}

		allEmbeddings = append(allEmbeddings, embRecord)
	}

	// Delete existing embeddings for this job (to handle updates)
	if err := uc.embeddingRepo.DeleteJobEmbeddings(ctx, job.ID); err != nil {
		uc.log.Warnf("failed to delete existing embeddings: %v", err)
		// Continue anyway - might be first time embedding
	}

	// Save all embeddings to vector database
	if err := uc.embeddingRepo.SaveJobEmbeddings(ctx, allEmbeddings); err != nil {
		uc.log.Errorf("failed to save embeddings: %v", err)
		return nil, fmt.Errorf("failed to save embeddings: %w", err)
	}

	uc.log.Infof("Successfully embedded job %s with %d vectors", job.ID, len(allEmbeddings))

	// Convert to result
	result := &JobEmbeddingResult{
		JobID:      job.ID,
		Embeddings: make([]*JobEmbeddingItem, len(allEmbeddings)),
	}

	for i, emb := range allEmbeddings {
		result.Embeddings[i] = &JobEmbeddingItem{
			ID:            emb.ID,
			EmbeddingType: string(emb.EmbeddingType),
			Embedding:     emb.Embedding,
			TextContent:   emb.TextContent,
		}
	}

	return result, nil
}

// EmbedJobPostingByID embeds a job posting by its ID
func (uc *JobEmbeddingUseCase) EmbedJobPostingByID(ctx context.Context, jobID string) (*JobEmbeddingResult, error) {
	// Get job from repository
	job, err := uc.jobRepo.GetJobPosting(ctx, jobID)
	if err != nil {
		uc.log.Errorf("failed to get job posting: %v", err)
		return nil, fmt.Errorf("failed to get job posting: %w", err)
	}

	if job == nil {
		return nil, ErrJobNotFound
	}

	return uc.EmbedJobPosting(ctx, job)
}

// SearchSimilarJobs searches for jobs similar to the given query
func (uc *JobEmbeddingUseCase) SearchSimilarJobs(ctx context.Context, query string, topK int) ([]*SimilarJobResult, error) {
	uc.log.WithContext(ctx).Infof("SearchSimilarJobs: query length=%d, topK=%d", len(query), topK)

	// Generate embedding for query
	queryEmbedding, err := uc.embeddingClient.CreateEmbedding(ctx, query)
	if err != nil {
		uc.log.Errorf("failed to create query embedding: %v", err)
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Search using full content embedding type
	results, err := uc.embeddingRepo.SearchSimilarJobs(ctx, queryEmbedding, string(milvusx.EmbeddingTypeFull), topK)
	if err != nil {
		uc.log.Errorf("failed to search similar jobs: %v", err)
		return nil, fmt.Errorf("failed to search similar jobs: %w", err)
	}

	uc.log.Infof("Found %d similar jobs", len(results))
	return results, nil
}

// SearchSimilarJobsBySkills searches for jobs with similar skills/tech stack
func (uc *JobEmbeddingUseCase) SearchSimilarJobsBySkills(ctx context.Context, skills []string, topK int) ([]*SimilarJobResult, error) {
	skillsText := strings.Join(skills, ", ")
	uc.log.WithContext(ctx).Infof("SearchSimilarJobsBySkills: %s, topK=%d", skillsText, topK)

	// Generate embedding for skills
	queryEmbedding, err := uc.embeddingClient.CreateEmbedding(ctx, skillsText)
	if err != nil {
		uc.log.Errorf("failed to create skills embedding: %v", err)
		return nil, fmt.Errorf("failed to create skills embedding: %w", err)
	}

	// Search using skills embedding type
	results, err := uc.embeddingRepo.SearchSimilarJobs(ctx, queryEmbedding, string(milvusx.EmbeddingTypeSkills), topK)
	if err != nil {
		uc.log.Errorf("failed to search similar jobs: %v", err)
		return nil, fmt.Errorf("failed to search similar jobs: %w", err)
	}

	uc.log.Infof("Found %d similar jobs by skills", len(results))
	return results, nil
}

// prepareEmbeddingTexts prepares text content for different embedding types
// This method now uses ontology-based normalization for better consistency
func (uc *JobEmbeddingUseCase) prepareEmbeddingTexts(job *JobPosting) map[string]string {
	texts := make(map[string]string)

	// Get the normalizer instance
	normalizer := GetDefaultNormalizer()

	// Normalize job data using ontology
	normalizedData := normalizer.NormalizeJobPosting(job)

	// Get normalized texts optimized for embedding
	normalizedTexts := normalizer.PrepareTextForEmbedding(normalizedData)

	// Full content embedding (normalized)
	texts[string(milvusx.EmbeddingTypeFull)] = normalizedTexts["full"]

	// Title embedding (normalized with category and seniority context)
	texts[string(milvusx.EmbeddingTypeTitle)] = normalizedTexts["title"]

	// Level embedding (normalized with years range and responsibility context)
	if job.Level != "" {
		texts[string(milvusx.EmbeddingTypeLevel)] = normalizedTexts["level"]
	}

	// Requirements embedding (normalized and categorized)
	if job.Requirements != "" {
		texts[string(milvusx.EmbeddingTypeRequirements)] = normalizedTexts["requirements"]
	}

	// Skills embedding (normalized with categories and related skills)
	if len(job.JobTech) > 0 {
		texts[string(milvusx.EmbeddingTypeSkills)] = normalizedTexts["skills"]
	}

	// Description embedding (keep original as it's usually free-form)
	if job.Description != "" {
		texts[string(milvusx.EmbeddingTypeDescription)] = job.Description
	}

	// Log normalization results for debugging
	uc.log.Debugf("Normalized job %s: Title[%s->%s] Level[%s->%s] Skills[%v->%v]",
		job.ID,
		normalizedData.OriginalTitle, normalizedData.NormalizedTitle,
		normalizedData.OriginalLevel, normalizedData.NormalizedLevel,
		normalizedData.OriginalSkills, normalizedData.NormalizedSkills)

	return texts
}

// generateEmbeddingID generates a unique ID for an embedding
func generateEmbeddingID(jobID string, embeddingType string) string {
	return fmt.Sprintf("%s_%s_%s", jobID, embeddingType, uuid.New().String()[:8])
}

// truncateText truncates text to a maximum character count
func truncateText(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars]
}

// ScoredJobResult represents a job with combined similarity score
type ScoredJobResult struct {
	JobID             string      `json:"job_id"`
	Job               *JobPosting `json:"job,omitempty"`
	TotalScore        float32     `json:"total_score"`
	TitleScore        float32     `json:"title_score"`
	RequirementsScore float32     `json:"requirements_score"`
	SkillsScore       float32     `json:"skills_score"`
	LevelScore        float32     `json:"level_score"`
}

// Weight configuration for scoring
const (
	WeightTitle        = 0.3
	WeightRequirements = 0.25
	WeightSkills       = 0.35
	WeightLevel        = 0.1
)

// FindSimilarJobsByJobID finds similar jobs based on a given job's embeddings
// It searches using title, requirements, skills, and level embeddings and combines scores
func (uc *JobEmbeddingUseCase) FindSimilarJobsByJobID(ctx context.Context, jobID string, topK int) ([]*ScoredJobResult, error) {
	uc.log.WithContext(ctx).Infof("FindSimilarJobsByJobID: jobID=%s, topK=%d", jobID, topK)

	// Define embedding types to use for similarity search
	embeddingTypes := []string{
		string(milvusx.EmbeddingTypeTitle),
		string(milvusx.EmbeddingTypeRequirements),
		string(milvusx.EmbeddingTypeSkills),
		string(milvusx.EmbeddingTypeLevel),
	}

	// Get embeddings for the source job
	jobEmbeddings, err := uc.embeddingRepo.GetJobEmbeddings(ctx, jobID, embeddingTypes)
	if err != nil {
		uc.log.Errorf("failed to get job embeddings: %v", err)
		return nil, fmt.Errorf("failed to get job embeddings: %w", err)
	}

	if len(jobEmbeddings) == 0 {
		return nil, fmt.Errorf("no embeddings found for job %s", jobID)
	}

	// Map to aggregate scores by job ID
	jobScores := make(map[string]*ScoredJobResult)

	// Search for each embedding type and aggregate scores
	for embType, embedding := range jobEmbeddings {
		if len(embedding) == 0 {
			continue
		}

		// Search similar jobs for this embedding type
		// Get more results to account for filtering
		results, err := uc.embeddingRepo.SearchSimilarJobs(ctx, embedding, embType, topK*3)
		if err != nil {
			uc.log.Warnf("failed to search similar jobs for %s: %v", embType, err)
			continue
		}

		// Get weight for this embedding type
		weight := getEmbeddingWeight(embType)

		// Aggregate scores
		for _, result := range results {
			// Skip the source job itself
			if result.JobID == jobID {
				continue
			}

			if _, exists := jobScores[result.JobID]; !exists {
				jobScores[result.JobID] = &ScoredJobResult{
					JobID: result.JobID,
				}
			}

			// Update specific scores
			switch embType {
			case string(milvusx.EmbeddingTypeTitle):
				jobScores[result.JobID].TitleScore = result.Score
			case string(milvusx.EmbeddingTypeRequirements):
				jobScores[result.JobID].RequirementsScore = result.Score
			case string(milvusx.EmbeddingTypeSkills):
				jobScores[result.JobID].SkillsScore = result.Score
			case string(milvusx.EmbeddingTypeLevel):
				jobScores[result.JobID].LevelScore = result.Score
			}

			// Add weighted score to total
			jobScores[result.JobID].TotalScore += result.Score * weight
		}
	}

	// Convert map to slice and sort by total score
	results := make([]*ScoredJobResult, 0, len(jobScores))
	for _, scored := range jobScores {
		results = append(results, scored)
	}

	// Sort by total score descending
	sortScoredResults(results)

	// Limit to topK
	if len(results) > topK {
		results = results[:topK]
	}

	// Optionally fetch job details for top results
	for _, scored := range results {
		job, err := uc.jobRepo.GetJobPosting(ctx, scored.JobID)
		if err == nil && job != nil {
			scored.Job = job
		}
	}

	uc.log.Infof("Found %d similar jobs for job %s", len(results), jobID)
	return results, nil
}

// getEmbeddingWeight returns the weight for a given embedding type
func getEmbeddingWeight(embType string) float32 {
	switch embType {
	case string(milvusx.EmbeddingTypeTitle):
		return WeightTitle
	case string(milvusx.EmbeddingTypeRequirements):
		return WeightRequirements
	case string(milvusx.EmbeddingTypeSkills):
		return WeightSkills
	case string(milvusx.EmbeddingTypeLevel):
		return WeightLevel
	default:
		return 0.0
	}
}

// sortScoredResults sorts results by total score in descending order
func sortScoredResults(results []*ScoredJobResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].TotalScore > results[i].TotalScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
