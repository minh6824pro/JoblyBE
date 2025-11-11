package server

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// WhiteListEntry represents a whitelist entry with method and path
type WhiteListEntry struct {
	Method string // GET, POST, PUT, DELETE, * (any method)
	Path   string
}

// NewWhiteListMatcher creates a new whitelist matcher
// Public endpoints that don't require authentication
func NewWhiteListMatcher() selector.MatchFunc {
	// Whitelist with method-specific rules
	whitelist := []WhiteListEntry{
		// Auth endpoints
		{Method: "POST", Path: "/api.auth.v1.Auth/Register"},
		{Method: "POST", Path: "/api.auth.v1.Auth/Login"},
		{Method: "POST", Path: "/api.auth.v1.Auth/RefreshToken"},

		// Job endpoints - public read access
		{Method: "GET", Path: "/api.job.v1.JobPosting/GetJobPosting"},
		{Method: "GET", Path: "/api.job.v1.JobPosting/ListJobPostings"},

		// Company endpoints - public read access
		{Method: "GET", Path: "/api.job.v1.Company/GetCompany"},
		{Method: "GET", Path: "/api.job.v1.Company/ListCompanies"},

		// Resume upload - public (authentication optional)
		//	{Method: "POST", Path: "/api/v1/resumes/upload"},

		// Swagger and health check - all methods
		{Method: "*", Path: "/q/swagger-ui"},
		{Method: "*", Path: "/q/openapi.json"},
		{Method: "*", Path: "/health"},
	}

	return func(ctx context.Context, operation string) bool {
		// Try to get HTTP method from transport context
		method := ""
		if tr, ok := transport.FromServerContext(ctx); ok {
			if ht, ok := tr.(http.Transporter); ok {
				method = ht.Request().Method
			}
		}

		// Check if operation matches whitelist
		for _, entry := range whitelist {
			if entry.Path == operation {
				// If method is "*", allow all methods
				if entry.Method == "*" {
					return false
				}
				// If method matches, allow
				if entry.Method == method {
					return false
				}
			}
		}

		// Apply middleware for all other cases
		return true
	}
}

// NewPathWhiteListMatcher creates a matcher based on HTTP path and method
// This is useful for HTTP transport where we want to check the actual path
func NewPathWhiteListMatcher() selector.MatchFunc {
	// Whitelist with method-specific rules for HTTP paths
	whitelist := []WhiteListEntry{
		// Auth endpoints
		{Method: "POST", Path: "/api/v1/auth/register"},
		{Method: "POST", Path: "/api/v1/auth/login"},
		{Method: "POST", Path: "/api/v1/auth/refresh-token"},

		// Job endpoints - public read access
		{Method: "GET", Path: "/api/v1/jobs"},  // List jobs
		{Method: "GET", Path: "/api/v1/jobs/"}, // Get specific job (with ID)

		// Company endpoints - public read access
		{Method: "GET", Path: "/api/v1/companies"},  // List companies
		{Method: "GET", Path: "/api/v1/companies/"}, // Get specific company (with ID)

		// Resume upload - public (authentication optional)
		{Method: "POST", Path: "/api/v1/resumes/upload"},

		// Swagger and health check - all methods
		{Method: "*", Path: "/q/swagger-ui"},
		{Method: "*", Path: "/q/openapi.json"},
		{Method: "*", Path: "/health"},
	}

	return func(ctx context.Context, operation string) bool {
		var path, method string

		// Try to get path and method from transport context
		if tr, ok := transport.FromServerContext(ctx); ok {
			if ht, ok := tr.(http.Transporter); ok {
				path = ht.Request().URL.Path
				method = ht.Request().Method
			}
		}

		// Check if path and method match whitelist
		for _, entry := range whitelist {
			// Exact match or prefix match (for paths with IDs)
			pathMatches := entry.Path == path || strings.HasPrefix(path, entry.Path)

			if pathMatches {
				// If method is "*", allow all methods
				if entry.Method == "*" {
					return false
				}
				// If method matches, allow
				if entry.Method == method {
					return false
				}
			}
		}

		// Check operation name as fallback
		for _, entry := range whitelist {
			if entry.Path == operation {
				if entry.Method == "*" || entry.Method == method {
					return false
				}
			}
		}

		// Apply middleware for all other cases
		return true
	}
}
