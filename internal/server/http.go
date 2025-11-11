package server

import (
	authv1 "JobblyBE/api/auth/v1"
	jobv1 "JobblyBE/api/job/v1"
	resumev1 "JobblyBE/api/resume/v1"

	"github.com/go-kratos/swagger-api/openapiv2"

	"JobblyBE/internal/conf"
	"JobblyBE/internal/service"
	"JobblyBE/pkg/middleware/auth"
	"JobblyBE/pkg/middleware/logging"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	cData *conf.Data,
	authSvc *service.AuthService,
	jobSvc *service.JobPostingService,
	companySvc *service.CompanyService,
	resumeSvc *service.ResumeService,
	logger log.Logger,
) *http.Server {
	// JWT secret from config
	jwtSecret := c.JwtSecret
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

	// Get resume parser URL from config
	resumeParserURL := c.ResumeParserUrl

	// Create upload handler
	uploadHandler, err := NewUploadHandler(resumeParserURL, logger, cData.Database.Source, cData.Database.Name, jwtSecret)
	if err != nil {
		panic(err)
	}
	// Register custom HTTP handlers
	// Resume upload endpoint (multipart/form-data)
	// Use HandleFunc for raw HTTP handler
	srv.HandleFunc("/api/v1/resumes/upload", uploadHandler.HandleUploadResume)

	// Register swagger ui url: http://<hostname>/q/swagger-ui/
	h := openapiv2.NewHandler()
	srv.HandlePrefix("/q/", h)
	return srv
}
