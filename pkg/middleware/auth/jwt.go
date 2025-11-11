package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenDuration  = 24 * time.Hour     // Access token expires in 1 hour
	RefreshTokenDuration = 7 * 24 * time.Hour // Refresh token expires in 7 days
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidTokenType = errors.New("invalid token type")
	ErrMissingClaims    = errors.New("missing claims in token")
)

// TokenType định nghĩa loại token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// JWTClaims chứa thông tin trong JWT token
type JWTClaims struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	FullName    string    `json:"full_name"`
	PhoneNumber string    `json:"phone_number"`
	Role        string    `json:"role"`
	TokenType   TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenPair chứa access token và refresh token
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// GenerateAccessToken tạo access token mới
func GenerateAccessToken(userID, email, fullName, phoneNumber, role, secret string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:      userID,
		Email:       email,
		FullName:    fullName,
		PhoneNumber: phoneNumber,
		Role:        role,
		TokenType:   AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "jobbly-auth-service",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken tạo refresh token mới
func GenerateRefreshToken(userID, email, fullName, phoneNumber, role, secret string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:      userID,
		Email:       email,
		FullName:    fullName,
		PhoneNumber: phoneNumber,
		Role:        role,
		TokenType:   RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "jobbly-auth-service",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// NewTokenPair tạo cả access token và refresh token
func NewTokenPair(userID, email, fullName, phoneNumber, role, secret string) (*TokenPair, error) {
	accessToken, err := GenerateAccessToken(userID, email, fullName, phoneNumber, role, secret, AccessTokenDuration)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateRefreshToken(userID, email, fullName, phoneNumber, role, secret, RefreshTokenDuration)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// ValidateToken validates và parse JWT token
func ValidateToken(tokenString, secret string, expectedTokenType TokenType) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Kiểm tra signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Kiểm tra token type
	if claims.TokenType != expectedTokenType {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrInvalidTokenType, expectedTokenType, claims.TokenType)
	}

	return claims, nil
}

// ValidateAccessToken validates access token
func ValidateAccessToken(tokenString, secret string) (*JWTClaims, error) {
	return ValidateToken(tokenString, secret, AccessToken)
}

// ValidateRefreshToken validates refresh token và trả về user ID
func ValidateRefreshToken(tokenString, secret string) (string, error) {
	claims, err := ValidateToken(tokenString, secret, RefreshToken)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// Context keys
type contextKey string

const claimsContextKey contextKey = "jwt_claims"

// SetClaimsToContext lưu JWT claims vào context
func SetClaimsToContext(ctx context.Context, claims *JWTClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// GetClaimsFromContext lấy JWT claims từ context
func GetClaimsFromContext(ctx context.Context) (*JWTClaims, error) {
	claims, ok := ctx.Value(claimsContextKey).(*JWTClaims)
	if !ok || claims == nil {
		return nil, ErrMissingClaims
	}
	return claims, nil
}
