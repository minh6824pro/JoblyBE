package biz

import (
	"JobblyBE/internal/conf"
	"JobblyBE/pkg/configx"
	"JobblyBE/pkg/openai"
	"context"
)

// embeddingClientImpl implements EmbeddingClient interface
type embeddingClientImpl struct {
	client *openai.Client
}

// NewEmbeddingClient creates a new embedding client
func NewEmbeddingClient(c *conf.Server) EmbeddingClient {
	apiKey := configx.GetEnvOrString("OPENAI_API_KEY", c.OpenaiApiKey)
	client := openai.NewClient(apiKey)
	return &embeddingClientImpl{
		client: client,
	}
}

// CreateEmbedding creates an embedding for a single text
func (c *embeddingClientImpl) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return c.client.CreateEmbedding(ctx, text)
}

// CreateEmbeddings creates embeddings for multiple texts
func (c *embeddingClientImpl) CreateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	return c.client.CreateEmbeddings(ctx, texts)
}
