package data

import (
	"JobblyBE/internal/conf"
	"JobblyBE/pkg/configx"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewUserRepo,
	NewJobPostingRepo,
	NewCompanyRepo,
	NewUserTrackingRepo,
	NewResumeRepo,
)

// Data .
type Data struct {
	db  *mongo.Database
	log *log.Helper
}

const (
	CollectionUser         = "user"
	CollectionCompany      = "company"
	CollectionJobPosting   = "job_posting"
	CollectionUserTracking = "user_tracking"
)

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {

	helper := log.NewHelper(logger)

	// Create MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(configx.GetEnvOrString("DATABASE_SOURCE", c.Database.Source)))
	if err != nil {
		helper.Errorf("failed to connect to mongodb: %v", err)
		return nil, nil, err
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		helper.Errorf("failed to ping mongodb: %v", err)
		return nil, nil, err
	}

	helper.Info("successfully connected to mongodb")

	// Get database name from config or use default
	dbName := configx.GetEnvOrString("DATABASE_NAME", c.Database.Name)

	db := client.Database(dbName)

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{db: db,
		log: helper}, cleanup, nil
}

// DB returns the MongoDB database instance
func (d *Data) DB() *mongo.Database {
	return d.db
}
