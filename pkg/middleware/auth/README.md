# JWT Authentication Middleware

This package provides JWT authentication middleware for the Jobbly backend service.

## Features

- JWT token generation (access token & refresh token)
- Token validation with expiration checking
- Role-based access control (RBAC)
- Context-based claims storage
- Kratos middleware integration

## Usage

### 1. Generate Token Pair

```go
import "JobblyBE/pkg/middleware/auth"

// Generate both access and refresh tokens
tokens, err := auth.NewTokenPair(
    userID,
    email,
    fullName,
    phoneNumber,
    role,
    jwtSecret,
)
if err != nil {
    // Handle error
}

// Use tokens
accessToken := tokens.AccessToken
refreshToken := tokens.RefreshToken
```

### 2. Apply JWT Middleware to HTTP Server

```go
import (
    "github.com/go-kratos/kratos/v2/transport/http"
    "JobblyBE/pkg/middleware/auth"
)

// Create HTTP server with JWT middleware
srv := http.NewServer(
    http.Middleware(
        recovery.Recovery(),
        auth.JWTAuth(jwtSecret), // Apply JWT middleware globally
    ),
)
```

### 3. Apply JWT Middleware to Specific Routes

```go
// Protect specific routes with JWT
authv1.RegisterAuthHTTPServer(srv, authSvc,
    // Skip JWT for public endpoints
    http.Middleware(
        selector.Server(
            auth.JWTAuth(jwtSecret),
        ).Match(NewWhiteListMatcher()).Build(),
    ),
)
```

### 4. Use Role-Based Access Control

```go
import "JobblyBE/pkg/middleware/auth"

// Only allow admin and moderator roles
srv := http.NewServer(
    http.Middleware(
        auth.JWTAuthWithRoles(jwtSecret, "admin", "moderator"),
    ),
)
```

### 5. Get User Claims in Service

```go
func (s *AuthService) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileReply, error) {
    // Get JWT claims from context
    claims, err := auth.GetClaimsFromContext(ctx)
    if err != nil {
        return nil, err
    }

    // Use claims
    userID := claims.UserID
    email := claims.Email
    role := claims.Role
    // ...
}
```

### 6. Validate Refresh Token

```go
// Validate refresh token and get user ID
userID, err := auth.ValidateRefreshToken(refreshToken, jwtSecret)
if err != nil {
    // Handle invalid or expired token
}
```

### 7. Generate New Access Token from Refresh Token

```go
// Generate new access token
newAccessToken, err := auth.GenerateAccessToken(
    userID,
    email,
    fullName,
    phoneNumber,
    role,
    jwtSecret,
    auth.AccessTokenDuration,
)
```

## Token Configuration

- **Access Token**: Expires in 15 minutes
- **Refresh Token**: Expires in 7 days

You can modify these durations in the `jwt.go` file:

```go
const (
    AccessTokenDuration  = 15 * time.Minute
    RefreshTokenDuration = 7 * 24 * time.Hour
)
```

## Token Format

Tokens are JWT tokens with the following claims:

```json
{
  "user_id": "string",
  "email": "string",
  "full_name": "string",
  "phone_number": "string",
  "role": "string",
  "token_type": "access|refresh",
  "exp": 1234567890,
  "iat": 1234567890,
  "nbf": 1234567890,
  "iss": "jobbly-auth-service",
  "sub": "user_id"
}
```

## Request Format

Clients should send the JWT token in the Authorization header:

```
Authorization: Bearer <access_token>
```

## Error Responses

The middleware returns the following errors:

- `AUTH_MISSING_TOKEN`: No token provided
- `AUTH_INVALID_HEADER`: Invalid Authorization header format
- `AUTH_TOKEN_INVALID`: Token validation failed
- `AUTH_TOKEN_EXPIRED`: Token has expired
- `AUTH_INSUFFICIENT_PERMISSION`: User doesn't have required role

## Security Best Practices

1. Store JWT secret in environment variable or secure config
2. Use HTTPS in production
3. Implement token refresh mechanism
4. Consider token blacklisting for logout
5. Rotate JWT secrets periodically
6. Use short expiration times for access tokens
