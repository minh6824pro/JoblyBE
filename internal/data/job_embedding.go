package data

import (
	"JobblyBE/internal/biz"
	"JobblyBE/pkg/milvusx"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type jobEmbeddingRepo struct {
	milvus *milvusx.Client
	log    *log.Helper
}

// NewJobEmbeddingRepo creates a new job embedding repository
func NewJobEmbeddingRepo(milvus *milvusx.Client, logger log.Logger) biz.JobEmbeddingRepo {
	return &jobEmbeddingRepo{
		milvus: milvus,
		log:    log.NewHelper(logger),
	}
}

// SaveJobEmbedding saves a single job embedding to Milvus
func (r *jobEmbeddingRepo) SaveJobEmbedding(ctx context.Context, jobID string, embeddingType string, embedding []float32, textContent string) error {
	embRecord := &milvusx.JobEmbedding{
		ID:            generateID(jobID, embeddingType),
		JobID:         jobID,
		EmbeddingType: milvusx.EmbeddingType(embeddingType),
		Embedding:     embedding,
		TextContent:   textContent,
		CreatedAt:     getCurrentTimestamp(),
	}

	if err := r.milvus.InsertJobEmbedding(ctx, embRecord); err != nil {
		r.log.Errorf("failed to save job embedding: %v", err)
		return err
	}

	// Flush to ensure data persistence
	if err := r.milvus.Flush(ctx); err != nil {
		r.log.Warnf("failed to flush after insert: %v", err)
	}

	return nil
}

// SaveJobEmbeddings saves multiple job embeddings to Milvus
func (r *jobEmbeddingRepo) SaveJobEmbeddings(ctx context.Context, embeddings []*milvusx.JobEmbedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	if err := r.milvus.InsertJobEmbeddings(ctx, embeddings); err != nil {
		r.log.Errorf("failed to save job embeddings: %v", err)
		return err
	}

	// Flush to ensure data persistence
	if err := r.milvus.Flush(ctx); err != nil {
		r.log.Warnf("failed to flush after insert: %v", err)
	}

	r.log.Infof("Saved %d job embeddings to Milvus", len(embeddings))
	return nil
}

// DeleteJobEmbeddings deletes all embeddings for a specific job
func (r *jobEmbeddingRepo) DeleteJobEmbeddings(ctx context.Context, jobID string) error {
	if err := r.milvus.DeleteJobEmbeddings(ctx, jobID); err != nil {
		r.log.Errorf("failed to delete job embeddings: %v", err)
		return err
	}
	return nil
}

// SearchSimilarJobs searches for similar jobs based on query vector
func (r *jobEmbeddingRepo) SearchSimilarJobs(ctx context.Context, queryVector []float32, embeddingType string, topK int) ([]*biz.SimilarJobResult, error) {
	results, err := r.milvus.SearchSimilarJobs(ctx, queryVector, milvusx.EmbeddingType(embeddingType), topK)
	if err != nil {
		r.log.Errorf("failed to search similar jobs: %v", err)
		return nil, err
	}

	// Convert to biz layer results
	bizResults := make([]*biz.SimilarJobResult, len(results))
	for i, result := range results {
		bizResults[i] = &biz.SimilarJobResult{
			JobID:       result.JobID,
			TextContent: result.TextContent,
			Score:       result.Score,
		}
	}

	return bizResults, nil
}

// GetJobEmbeddings retrieves embeddings for a specific job by embedding types
func (r *jobEmbeddingRepo) GetJobEmbeddings(ctx context.Context, jobID string, embeddingTypes []string) (map[string][]float32, error) {
	embeddings, err := r.milvus.GetJobEmbeddings(ctx, jobID, embeddingTypes)
	if err != nil {
		r.log.Errorf("failed to get job embeddings: %v", err)
		return nil, err
	}

	r.log.Infof("Retrieved %d embeddings for job %s", len(embeddings), jobID)
	return embeddings, nil
}

// Helper functions

func generateID(jobID string, embeddingType string) string {
	return jobID + "_" + embeddingType
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
