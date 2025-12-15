package data

import (
	"JobblyBE/internal/conf"
	"JobblyBE/pkg/configx"
	"JobblyBE/pkg/kafkax"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewKafkaProducer,
	NewUserRepo,
	NewJobPostingRepo,
	NewCompanyRepo,
	NewUserTrackingRepo,
	NewResumeRepo,
	NewJobApplicationRepo,
	NewNotificationRepo,
)

// Data .
type Data struct {
	db  *mongo.Database
	log *log.Helper
}

const (
	CollectionUser         = "users"
	CollectionCompany      = "companies"
	CollectionJobPosting   = "job_postings"
	CollectionUserTracking = "user_tracking"
	CollectionApplication  = "job_applications"
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

	data := &Data{db: db,
		log: helper}
	data.InitAllIndexes(ctx)
	return data, cleanup, nil
}

// DB returns the MongoDB database instance
func (d *Data) DB() *mongo.Database {
	return d.db
}

// GetCollection returns a collection by name
func (d *Data) GetCollection(name string) *mongo.Collection {
	return d.db.Collection(name)
}

// initUserIndexes creates indexes for users collection
func (d *Data) initUserIndexes(ctx context.Context, collectionName string) {
	col := d.GetCollection(collectionName)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetName("idx_email").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "role", Value: 1}},
			Options: options.Index().SetName("idx_role"),
		},
		{
			Keys:    bson.D{{Key: "active", Value: 1}},
			Options: options.Index().SetName("idx_active"),
		},
		{
			Keys:    bson.D{{Key: "saved_jd_id", Value: 1}},
			Options: options.Index().SetName("idx_saved_jd_id"),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		d.log.Warnf("failed to create indexes for %s: %v", collectionName, err)
	} else {
		d.log.Infof("Created indexes for collection: %s", collectionName)
	}
}

// initCompanyIndexes creates indexes for companies collection
func (d *Data) initCompanyIndexes(ctx context.Context, collectionName string) {
	col := d.GetCollection(collectionName)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetName("idx_name"),
		},
		{
			Keys:    bson.D{{Key: "industry", Value: 1}},
			Options: options.Index().SetName("idx_industry"),
		},
		{
			Keys:    bson.D{{Key: "location", Value: 1}},
			Options: options.Index().SetName("idx_location"),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		d.log.Warnf("failed to create indexes for %s: %v", collectionName, err)
	} else {
		d.log.Infof("Created indexes for collection: %s", collectionName)
	}
}

// initJobPostingIndexes creates indexes for job_postings collection
func (d *Data) initJobPostingIndexes(ctx context.Context, collectionName string) {
	col := d.GetCollection(collectionName)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "company_id", Value: 1}},
			Options: options.Index().SetName("idx_company_id"),
		},
		{
			Keys:    bson.D{{Key: "title", Value: "text"}},
			Options: options.Index().SetName("idx_title_text"),
		},
		{
			Keys:    bson.D{{Key: "level", Value: 1}},
			Options: options.Index().SetName("idx_level"),
		},
		{
			Keys:    bson.D{{Key: "job_type", Value: 1}},
			Options: options.Index().SetName("idx_job_type"),
		},
		{
			Keys:    bson.D{{Key: "location", Value: 1}},
			Options: options.Index().SetName("idx_location"),
		},
		{
			Keys:    bson.D{{Key: "job_tech", Value: 1}},
			Options: options.Index().SetName("idx_job_tech"),
		},
		{
			Keys:    bson.D{{Key: "posted_at", Value: -1}},
			Options: options.Index().SetName("idx_posted_at"),
		},
		{
			Keys:    bson.D{{Key: "active", Value: 1}},
			Options: options.Index().SetName("idx_active"),
		},
		{
			Keys: bson.D{
				{Key: "company_id", Value: 1},
				{Key: "posted_at", Value: -1},
			},
			Options: options.Index().SetName("idx_company_posted"),
		},
		{
			Keys: bson.D{
				{Key: "company_id", Value: 1},
				{Key: "active", Value: 1},
			},
			Options: options.Index().SetName("idx_company_active"),
		},
		{
			Keys: bson.D{
				{Key: "active", Value: 1},
				{Key: "posted_at", Value: -1},
			},
			Options: options.Index().SetName("idx_active_posted"),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		d.log.Warnf("failed to create indexes for %s: %v", collectionName, err)
	} else {
		d.log.Infof("Created indexes for collection: %s", collectionName)
	}
}

// initApplicationIndexes creates indexes for job_applications collection
func (d *Data) initApplicationIndexes(ctx context.Context, collectionName string) {
	col := d.GetCollection(collectionName)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_user_id"),
		},
		{
			Keys:    bson.D{{Key: "job_id", Value: 1}},
			Options: options.Index().SetName("idx_job_id"),
		},
		{
			Keys:    bson.D{{Key: "resume_id", Value: 1}},
			Options: options.Index().SetName("idx_resume_id"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "applied_at", Value: -1}},
			Options: options.Index().SetName("idx_applied_at"),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "job_id", Value: 1},
			},
			Options: options.Index().SetName("idx_user_job_unique").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "job_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("idx_job_status"),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		d.log.Warnf("failed to create indexes for %s: %v", collectionName, err)
	} else {
		d.log.Infof("Created indexes for collection: %s", collectionName)
	}
}

// initUserTrackingIndexes creates indexes for user_tracking collection
func (d *Data) initUserTrackingIndexes(ctx context.Context, collectionName string) {
	col := d.GetCollection(collectionName)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_user_id"),
		},
		{
			Keys:    bson.D{{Key: "tracking_type", Value: 1}},
			Options: options.Index().SetName("idx_tracking_type"),
		},
		{
			Keys:    bson.D{{Key: "metadata.job_id", Value: 1}},
			Options: options.Index().SetName("idx_metadata_job_id"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "tracking_type", Value: 1},
				{Key: "metadata.job_id", Value: 1},
			},
			Options: options.Index().SetName("idx_user_tracking_job"),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "metadata.time_on_sight", Value: -1},
			},
			Options: options.Index().SetName("idx_user_time_on_sight"),
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		d.log.Warnf("failed to create indexes for %s: %v", collectionName, err)
	} else {
		d.log.Infof("Created indexes for collection: %s", collectionName)
	}
}

// InitAllIndexes initializes indexes for all collections
func (d *Data) InitAllIndexes(ctx context.Context) error {
	d.log.Info("Initializing database indexes...")

	d.initUserIndexes(ctx, CollectionUser)
	d.initCompanyIndexes(ctx, CollectionCompany)
	d.initJobPostingIndexes(ctx, CollectionJobPosting)
	d.initApplicationIndexes(ctx, CollectionApplication)
	d.initUserTrackingIndexes(ctx, CollectionUserTracking)
	d.initNotificationIndexes(ctx, "notifications")

	d.log.Info("Database indexes initialization completed")
	return nil
}

// initNotificationIndexes creates indexes for notifications collection
func (d *Data) initNotificationIndexes(ctx context.Context, collectionName string) {
	col := d.GetCollection(collectionName)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_user_id"),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "is_read", Value: 1},
			},
			Options: options.Index().SetName("idx_user_id_is_read"),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_user_id_created_at"),
		},
	}

	for _, index := range indexes {
		_, err := col.Indexes().CreateOne(ctx, index)
		if err != nil {
			d.log.Warnf("Failed to create index %s on %s: %v", *index.Options.Name, collectionName, err)
		} else {
			d.log.Infof("Created index %s on %s", *index.Options.Name, collectionName)
		}
	}
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(cKafka *conf.Kafka, logger log.Logger) *kafkax.Producer {
	kafkaBrokers := getKafkaBrokers(cKafka)
	kafkaTimeout := 30 * time.Second
	if cKafka.Timeout != nil {
		kafkaTimeout = cKafka.Timeout.AsDuration()
	}

	producerConfig := &kafkax.ProducerConfig{
		Brokers:    kafkaBrokers,
		Username:   configx.GetEnvOrString("KAFKA_USERNAME", cKafka.Username),
		Password:   configx.GetEnvOrString("KAFKA_PASSWORD", cKafka.Password),
		EnableSASL: cKafka.EnableSasl,
		Timeout:    kafkaTimeout,
	}

	return kafkax.NewProducer(producerConfig, logger)
}

func getKafkaBrokers(cKafka *conf.Kafka) []string {
	envBrokers := configx.GetEnvOrString("KAFKA_BROKERS", "")
	if envBrokers != "" {
		return []string{envBrokers}
	}
	if len(cKafka.Brokers) > 0 {
		return cKafka.Brokers
	}
	return []string{"localhost:9092"}
}
