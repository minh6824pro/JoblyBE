package milvusx

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	// Collection names
	CollectionJobEmbedding = "job_embeddings"

	// Embedding dimensions
	EmbeddingDimension = 1536 // OpenAI text-embedding-3-small dimension

	// Field names
	FieldID            = "id"
	FieldJobID         = "job_id"
	FieldEmbeddingType = "embedding_type"
	FieldEmbedding     = "embedding"
	FieldTextContent   = "text_content"
	FieldCreatedAt     = "created_at"

	// Index parameters
	IndexName = "embedding_index"
	IndexType = "IVF_FLAT"
	MetricL2  = "L2"
	MetricIP  = "IP" // Inner Product - better for normalized embeddings
	NList     = 128
)

// EmbeddingType defines the type of content being embedded
type EmbeddingType string

const (
	EmbeddingTypeFull         EmbeddingType = "full"         // Full job description
	EmbeddingTypeTitle        EmbeddingType = "title"        // Job title only
	EmbeddingTypeRequirements EmbeddingType = "requirements" // Requirements only
	EmbeddingTypeSkills       EmbeddingType = "skills"       // Skills/tech stack
	EmbeddingTypeDescription  EmbeddingType = "description"  // Description only
	EmbeddingTypeLevel        EmbeddingType = "level"        // Job level (ENTRY, JUNIOR, MID, SENIOR, LEAD)
)

// Config holds Milvus connection configuration
type Config struct {
	Address  string
	Username string
	Password string
	DBName   string
}

// Client wraps the Milvus client
type Client struct {
	client client.Client
	log    *log.Helper
}

// NewClient creates a new Milvus client
func NewClient(cfg *Config, logger log.Logger) (*Client, func(), error) {
	helper := log.NewHelper(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var c client.Client
	var err error

	if cfg.Username != "" && cfg.Password != "" {
		c, err = client.NewClient(ctx, client.Config{
			Address:  cfg.Address,
			Username: cfg.Username,
			Password: cfg.Password,
			DBName:   cfg.DBName,
		})
	} else {
		c, err = client.NewClient(ctx, client.Config{
			Address: cfg.Address,
			DBName:  cfg.DBName,
		})
	}

	if err != nil {
		helper.Errorf("failed to connect to Milvus: %v", err)
		return nil, nil, err
	}

	helper.Info("successfully connected to Milvus")

	cleanup := func() {
		helper.Info("closing Milvus connection")
		c.Close()
	}

	milvusClient := &Client{
		client: c,
		log:    helper,
	}

	// Initialize collections
	if err := milvusClient.InitCollections(ctx); err != nil {
		helper.Errorf("failed to initialize Milvus collections: %v", err)
		return nil, nil, err
	}

	return milvusClient, cleanup, nil
}

// InitCollections initializes all required collections with schemas
func (c *Client) InitCollections(ctx context.Context) error {
	c.log.Info("Initializing Milvus collections...")

	// Initialize job embeddings collection
	if err := c.initJobEmbeddingCollection(ctx); err != nil {
		return err
	}

	c.log.Info("Milvus collections initialization completed")
	return nil
}

// initJobEmbeddingCollection creates the job_embeddings collection if not exists
func (c *Client) initJobEmbeddingCollection(ctx context.Context) error {
	collectionName := CollectionJobEmbedding

	// Check if collection exists
	exists, err := c.client.HasCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		c.log.Infof("Collection %s already exists", collectionName)

		// Load collection for search
		err = c.client.LoadCollection(ctx, collectionName, false)
		if err != nil {
			c.log.Warnf("failed to load collection: %v", err)
		}
		return nil
	}

	// Define schema
	schema := &entity.Schema{
		CollectionName: collectionName,
		Description:    "Job posting embeddings for semantic search",
		AutoID:         false,
		Fields: []*entity.Field{
			{
				Name:       FieldID,
				DataType:   entity.FieldTypeVarChar,
				PrimaryKey: true,
				AutoID:     false,
				TypeParams: map[string]string{
					"max_length": "128",
				},
			},
			{
				Name:     FieldJobID,
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     FieldEmbeddingType,
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "32",
				},
			},
			{
				Name:     FieldEmbedding,
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", EmbeddingDimension),
				},
			},
			{
				Name:     FieldTextContent,
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     FieldCreatedAt,
				DataType: entity.FieldTypeInt64,
			},
		},
	}

	// Create collection
	err = c.client.CreateCollection(ctx, schema, 2) // 2 shards
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
	}

	c.log.Infof("Created collection: %s", collectionName)

	// Create index on embedding field
	idx, err := entity.NewIndexIvfFlat(entity.IP, NList)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	err = c.client.CreateIndex(ctx, collectionName, FieldEmbedding, idx, false)
	if err != nil {
		return fmt.Errorf("failed to create index on %s: %w", FieldEmbedding, err)
	}

	c.log.Infof("Created index on %s.%s", collectionName, FieldEmbedding)

	// Load collection for search
	err = c.client.LoadCollection(ctx, collectionName, false)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	c.log.Infof("Loaded collection: %s", collectionName)

	return nil
}

// GetClient returns the underlying Milvus client
func (c *Client) GetClient() client.Client {
	return c.client
}

// JobEmbedding represents a job embedding record
type JobEmbedding struct {
	ID            string
	JobID         string
	EmbeddingType EmbeddingType
	Embedding     []float32
	TextContent   string
	CreatedAt     int64
}

// InsertJobEmbedding inserts a job embedding into Milvus
func (c *Client) InsertJobEmbedding(ctx context.Context, embedding *JobEmbedding) error {
	// Prepare data columns
	idColumn := entity.NewColumnVarChar(FieldID, []string{embedding.ID})
	jobIDColumn := entity.NewColumnVarChar(FieldJobID, []string{embedding.JobID})
	embeddingTypeColumn := entity.NewColumnVarChar(FieldEmbeddingType, []string{string(embedding.EmbeddingType)})
	embeddingColumn := entity.NewColumnFloatVector(FieldEmbedding, EmbeddingDimension, [][]float32{embedding.Embedding})
	textContentColumn := entity.NewColumnVarChar(FieldTextContent, []string{embedding.TextContent})
	createdAtColumn := entity.NewColumnInt64(FieldCreatedAt, []int64{embedding.CreatedAt})

	_, err := c.client.Insert(ctx, CollectionJobEmbedding, "",
		idColumn, jobIDColumn, embeddingTypeColumn, embeddingColumn, textContentColumn, createdAtColumn)
	if err != nil {
		return fmt.Errorf("failed to insert job embedding: %w", err)
	}

	c.log.Infof("Inserted job embedding: job_id=%s, type=%s", embedding.JobID, embedding.EmbeddingType)
	return nil
}

// InsertJobEmbeddings inserts multiple job embeddings into Milvus
func (c *Client) InsertJobEmbeddings(ctx context.Context, embeddings []*JobEmbedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	ids := make([]string, len(embeddings))
	jobIDs := make([]string, len(embeddings))
	embeddingTypes := make([]string, len(embeddings))
	vectors := make([][]float32, len(embeddings))
	textContents := make([]string, len(embeddings))
	createdAts := make([]int64, len(embeddings))

	for i, emb := range embeddings {
		ids[i] = emb.ID
		jobIDs[i] = emb.JobID
		embeddingTypes[i] = string(emb.EmbeddingType)
		vectors[i] = emb.Embedding
		textContents[i] = emb.TextContent
		createdAts[i] = emb.CreatedAt
	}

	idColumn := entity.NewColumnVarChar(FieldID, ids)
	jobIDColumn := entity.NewColumnVarChar(FieldJobID, jobIDs)
	embeddingTypeColumn := entity.NewColumnVarChar(FieldEmbeddingType, embeddingTypes)
	embeddingColumn := entity.NewColumnFloatVector(FieldEmbedding, EmbeddingDimension, vectors)
	textContentColumn := entity.NewColumnVarChar(FieldTextContent, textContents)
	createdAtColumn := entity.NewColumnInt64(FieldCreatedAt, createdAts)

	_, err := c.client.Insert(ctx, CollectionJobEmbedding, "",
		idColumn, jobIDColumn, embeddingTypeColumn, embeddingColumn, textContentColumn, createdAtColumn)
	if err != nil {
		return fmt.Errorf("failed to insert job embeddings: %w", err)
	}

	c.log.Infof("Inserted %d job embeddings", len(embeddings))
	return nil
}

// SearchSimilarJobs searches for similar jobs based on embedding vector
func (c *Client) SearchSimilarJobs(ctx context.Context, queryVector []float32, embeddingType EmbeddingType, topK int) ([]*SearchResult, error) {
	// Build search parameters
	sp, err := entity.NewIndexIvfFlatSearchParam(16) // nprobe
	if err != nil {
		return nil, fmt.Errorf("failed to create search param: %w", err)
	}

	// Build expression for filtering by embedding type
	expr := fmt.Sprintf("%s == \"%s\"", FieldEmbeddingType, string(embeddingType))

	// Search
	results, err := c.client.Search(ctx, CollectionJobEmbedding,
		nil,
		expr,
		[]string{FieldJobID, FieldTextContent, FieldCreatedAt},
		[]entity.Vector{entity.FloatVector(queryVector)},
		FieldEmbedding,
		entity.IP,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar jobs: %w", err)
	}

	var searchResults []*SearchResult
	for _, result := range results {
		for i := 0; i < result.ResultCount; i++ {
			jobIDCol, _ := result.Fields.GetColumn(FieldJobID).(*entity.ColumnVarChar)
			textContentCol, _ := result.Fields.GetColumn(FieldTextContent).(*entity.ColumnVarChar)

			searchResults = append(searchResults, &SearchResult{
				JobID:       jobIDCol.Data()[i],
				TextContent: textContentCol.Data()[i],
				Score:       result.Scores[i],
			})
		}
	}

	return searchResults, nil
}

// SearchResult represents a search result
type SearchResult struct {
	JobID       string
	TextContent string
	Score       float32
}

// DeleteJobEmbeddings deletes all embeddings for a specific job
func (c *Client) DeleteJobEmbeddings(ctx context.Context, jobID string) error {
	expr := fmt.Sprintf("%s == \"%s\"", FieldJobID, jobID)

	err := c.client.Delete(ctx, CollectionJobEmbedding, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete job embeddings: %w", err)
	}

	c.log.Infof("Deleted embeddings for job: %s", jobID)
	return nil
}

// Flush flushes the collection to ensure data persistence
func (c *Client) Flush(ctx context.Context) error {
	err := c.client.Flush(ctx, CollectionJobEmbedding, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection: %w", err)
	}
	return nil
}

// GetJobEmbeddings retrieves embeddings for a specific job by embedding types
func (c *Client) GetJobEmbeddings(ctx context.Context, jobID string, embeddingTypes []string) (map[string][]float32, error) {
	// Build expression to filter by job_id and embedding_types
	var typeConditions []string
	for _, t := range embeddingTypes {
		typeConditions = append(typeConditions, fmt.Sprintf("%s == \"%s\"", FieldEmbeddingType, t))
	}

	expr := fmt.Sprintf("%s == \"%s\"", FieldJobID, jobID)
	if len(typeConditions) > 0 {
		expr = fmt.Sprintf("(%s) && (%s)", expr, strings.Join(typeConditions, " || "))
	}

	// Query to get embeddings
	result, err := c.client.Query(ctx, CollectionJobEmbedding, nil, expr,
		[]string{FieldEmbeddingType, FieldEmbedding})
	if err != nil {
		return nil, fmt.Errorf("failed to query job embeddings: %w", err)
	}

	embeddings := make(map[string][]float32)

	// Extract embedding type column
	embTypeCol, ok := result.GetColumn(FieldEmbeddingType).(*entity.ColumnVarChar)
	if !ok {
		return nil, fmt.Errorf("failed to get embedding type column")
	}

	// Extract embedding vector column
	embCol, ok := result.GetColumn(FieldEmbedding).(*entity.ColumnFloatVector)
	if !ok {
		return nil, fmt.Errorf("failed to get embedding column")
	}

	// Map embeddings by type
	for i := 0; i < embTypeCol.Len(); i++ {
		embType := embTypeCol.Data()[i]
		vector := embCol.Data()[i]
		embeddings[embType] = vector
	}

	c.log.Infof("Retrieved %d embeddings for job %s", len(embeddings), jobID)
	return embeddings, nil
}
