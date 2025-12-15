package server

import (
	applicationv1 "JobblyBE/api/application/v1"
	authv1 "JobblyBE/api/auth/v1"
	jobv1 "JobblyBE/api/job/v1"
	notificationv1 "JobblyBE/api/notification/v1"
	resumev1 "JobblyBE/api/resume/v1"
	userv1 "JobblyBE/api/user/v1"
	"JobblyBE/internal/biz"
	"JobblyBE/internal/conf"
	"JobblyBE/internal/data"
	"JobblyBE/internal/service"
	"JobblyBE/pkg/configx"
	"JobblyBE/pkg/kafkax"
	"JobblyBE/pkg/middleware/auth"
	"JobblyBE/pkg/middleware/logging"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/swagger-api/openapiv2"
	"github.com/gorilla/handlers"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	cData *conf.Data,
	cKafka *conf.Kafka,
	authSvc *service.AuthService,
	jobSvc *service.JobPostingService,
	companySvc *service.CompanyService,
	resumeSvc *service.ResumeService,
	userSvc *service.UserService,
	applicationSvc *service.JobApplicationService,
	notificationSvc *service.NotificationService,
	dataLayer *data.Data,
	resumeUC *biz.ResumeUseCase,
	notificationUC *biz.NotificationUseCase,
	applicationUC *biz.JobApplicationUseCase,
	logger log.Logger,
) *http.Server {
	// JWT secret from config
	jwtSecret := configx.GetEnvOrString("JWT_SECRET", c.JwtSecret)

	if jwtSecret == "" {
		log.Fatal("JWT secret is not configured")
	}

	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
			// Layer 1: Always parse JWT if present (for all endpoints)
			auth.OptionalJWTAuth(jwtSecret),
			// Layer 2: Require valid JWT (only for protected endpoints)
			selector.Server(
				auth.JWTAuth(jwtSecret),
			).Match(NewWhiteListMatcher()).Build(),
		),
		http.Filter(handlers.CORS(
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}),
			handlers.AllowedHeaders([]string{"Accept", "Api-Version", "Authorization", "Ocp-Apim-Subscription-Key", "Referer", "User-Agent", "Content-Type", "X-Platform", "X-App-Id", "X-TimeZone", "X-Timezone", "X-Locale"}),
			handlers.AllowedOrigins([]string{"*"}),
		)),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	authv1.RegisterAuthHTTPServer(srv, authSvc)
	jobv1.RegisterJobPostingHTTPServer(srv, jobSvc)
	jobv1.RegisterCompanyHTTPServer(srv, companySvc)
	resumev1.RegisterResumeHTTPServer(srv, resumeSvc)
	userv1.RegisterUserHTTPServer(srv, userSvc)
	applicationv1.RegisterJobApplicationHTTPServer(srv, applicationSvc)
	notificationv1.RegisterNotificationHTTPServer(srv, notificationSvc)

	// Get config values
	resumeParserURL := configx.GetEnvOrString("RESUME_PARSER_URL", c.ResumeParserUrl)
	databaseSource := configx.GetEnvOrString("DATABASE_SOURCE", cData.Database.Source)
	databaseName := configx.GetEnvOrString("DATABASE_NAME", cData.Database.Name)

	// Create WebSocket hub
	hub := NewHub(logger)
	go hub.Run()

	// Wrap hub with adapter to match biz.WebSocketHub interface
	hubAdapter := NewHubAdapter(hub)

	// Inject hub into application use case for sending notifications
	applicationUC.SetNotificationDependencies(notificationUC, hubAdapter)

	// Create Kafka producer
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
	producer := kafkax.NewProducer(producerConfig, logger)

	// Create resume parse job repository
	jobRepo := data.NewResumeParseJobRepo(dataLayer, logger)

	// Create upload handler config
	uploadConfig := &UploadHandlerConfig{
		ParserURL:      resumeParserURL,
		JwtSecret:      jwtSecret,
		DatabaseSource: databaseSource,
		DatabaseName:   databaseName,
		KafkaTopic:     configx.GetEnvOrString("KAFKA_RESUME_PARSE_TOPIC", cKafka.ResumeParseTopic),
	}

	// Create upload handler
	uploadHandler, err := NewUploadHandler(uploadConfig, producer, jobRepo, logger)
	if err != nil {
		panic(err)
	}

	// Create WebSocket handler
	wsHandler := NewWebSocketHandler(hub, jwtSecret, logger)

	// Create and start resume worker
	workerConfig := &ResumeWorkerConfig{
		Brokers:       kafkaBrokers,
		Username:      configx.GetEnvOrString("KAFKA_USERNAME", cKafka.Username),
		Password:      configx.GetEnvOrString("KAFKA_PASSWORD", cKafka.Password),
		EnableSASL:    cKafka.EnableSasl,
		Timeout:       kafkaTimeout,
		Topic:         configx.GetEnvOrString("KAFKA_RESUME_PARSE_TOPIC", cKafka.ResumeParseTopic),
		GroupID:       configx.GetEnvOrString("KAFKA_CONSUMER_GROUP", cKafka.ConsumerGroup),
		NumPartitions: int(cKafka.NumPartitions),
		NumWorkers:    int(cKafka.NumWorkers),
		ParserURL:     resumeParserURL,
	}

	resumeWorker := NewResumeWorker(workerConfig, producer, hub, jobRepo, dataLayer.DB(), logger)
	go resumeWorker.Start()

	// Create and start evaluate worker
	evaluateWorkerConfig := &EvaluateWorkerConfig{
		Brokers:       kafkaBrokers,
		Username:      configx.GetEnvOrString("KAFKA_USERNAME", cKafka.Username),
		Password:      configx.GetEnvOrString("KAFKA_PASSWORD", cKafka.Password),
		EnableSASL:    cKafka.EnableSasl,
		Timeout:       kafkaTimeout,
		Topic:         "evaluate", // Topic for resume evaluation
		GroupID:       configx.GetEnvOrString("KAFKA_CONSUMER_GROUP", cKafka.ConsumerGroup) + "-evaluate",
		NumPartitions: int(cKafka.NumPartitions),
		NumWorkers:    int(cKafka.NumWorkers),
	}

	evaluateWorker := NewEvaluateWorker(evaluateWorkerConfig, resumeUC, notificationUC, hub, logger)
	go evaluateWorker.Start()

	// Register custom HTTP handlers
	// Note: CORS is automatically handled by the handlers.CORS filter above
	// No need to wrap individual handlers

	// Resume upload endpoint (async - via Kafka)
	srv.HandleFunc("/api/v1/resumes/upload", uploadHandler.HandleUploadResumeAsync)
	// Resume upload endpoint (sync - backward compatibility)
	srv.HandleFunc("/api/v1/resumes/upload/sync", uploadHandler.HandleUploadResume)
	// Get job status endpoint
	srv.HandleFunc("/api/v1/resumes/job-status", uploadHandler.HandleGetJobStatus)
	// WebSocket endpoint for real-time updates
	srv.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// Register swagger ui with custom endpoints: http://<hostname>/q/swagger-ui/
	h := openapiv2.NewHandler()
	srv.HandlePrefix("/q/", h)
	return srv
}

// getKafkaBrokers returns Kafka brokers from config or environment
func getKafkaBrokers(cKafka *conf.Kafka) []string {
	envBrokers := configx.GetEnvOrString("KAFKA_BROKERS", "")
	if envBrokers != "" {
		return strings.Split(envBrokers, ",")
	}
	if len(cKafka.Brokers) > 0 {
		return cKafka.Brokers
	}
	return []string{"localhost:9092"}
}
