package auth

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	ErrMissingToken      = errors.Unauthorized("AUTH_MISSING_TOKEN", "missing authentication token")
	ErrInvalidAuthHeader = errors.Unauthorized("AUTH_INVALID_HEADER", "invalid authorization header format")
	ErrTokenValidation   = errors.Unauthorized("AUTH_TOKEN_INVALID", "token validation failed")
	ErrTokenExpired      = errors.Unauthorized("AUTH_TOKEN_EXPIRED", "token has expired")
)

// JWTAuth returns a JWT authentication middleware
func JWTAuth(secret string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// Check if claims already set by OptionalJWTAuth
			claims, err := GetClaimsFromContext(ctx)
			if err == nil && claims != nil {
				// Claims already validated - continue
				return handler(ctx, req)
			}

			// No claims found - token is required for this endpoint
			// Lấy token từ header
			token, tokenErr := extractToken(ctx)
			if tokenErr != nil {
				return nil, tokenErr
			}

			// Validate token
			claims, err = ValidateAccessToken(token, secret)
			if err != nil {
				if err == ErrExpiredToken {
					return nil, ErrTokenExpired
				}
				return nil, errors.Unauthorized("AUTH_TOKEN_INVALID", err.Error())
			}

			// Lưu claims vào context để sử dụng trong service
			ctx = SetClaimsToContext(ctx, claims)

			// Continue to next handler
			return handler(ctx, req)
		}
	}
}

// JWTAuthWithRoles returns a JWT authentication middleware with role-based access control
func JWTAuthWithRoles(secret string, allowedRoles ...string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// Lấy token từ header
			token, err := extractToken(ctx)
			if err != nil {
				return nil, err
			}

			// Validate token
			claims, err := ValidateAccessToken(token, secret)
			if err != nil {
				if err == ErrExpiredToken {
					return nil, ErrTokenExpired
				}
				return nil, errors.Unauthorized("AUTH_TOKEN_INVALID", err.Error())
			}

			// Kiểm tra role nếu có chỉ định
			if len(allowedRoles) > 0 {
				hasValidRole := false
				for _, role := range allowedRoles {
					if claims.Role == role {
						hasValidRole = true
						break
					}
				}
				if !hasValidRole {
					return nil, errors.Forbidden("AUTH_INSUFFICIENT_PERMISSION", "insufficient permissions for this resource")
				}
			}

			// Lưu claims vào context để sử dụng trong service
			ctx = SetClaimsToContext(ctx, claims)

			// Continue to next handler
			return handler(ctx, req)
		}
	}
}

// extractToken extracts JWT token from Authorization header
func extractToken(ctx context.Context) (string, error) {
	// Get transport info from context
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "", ErrMissingToken
	}

	// Get Authorization header
	authHeader := tr.RequestHeader().Get("Authorization")
	if authHeader == "" {
		return "", ErrMissingToken
	}

	// Parse Bearer token
	// Expected format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", ErrInvalidAuthHeader
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", ErrMissingToken
	}

	return token, nil
}

// OptionalJWTAuth returns a middleware that parses JWT if present but doesn't require it
// This allows both authenticated and anonymous users to access the same endpoint
// If token is present and valid, claims will be set to context
// If token is missing or invalid, request continues without claims
func OptionalJWTAuth(secret string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// Try to extract token (don't fail if missing)
			token, err := extractToken(ctx)
			if err != nil {
				// No token or invalid header format - continue without authentication
				return handler(ctx, req)
			}

			// Try to validate token (don't fail if invalid)
			claims, err := ValidateAccessToken(token, secret)
			if err == nil {
				// Token is valid - set claims to context
				ctx = SetClaimsToContext(ctx, claims)
			}
			// If token is invalid, just continue without claims

			// Continue to next handler
			return handler(ctx, req)
		}
	}
}
