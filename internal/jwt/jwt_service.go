package jwt

import (
	entities "Jobly/internal/entities"
	"Jobly/internal/errors"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"os"
	"time"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type JWTClaims struct {
	UserID uint          `json:"user_id"`
	Email  string        `json:"email"`
	Role   entities.Role `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct{}

func NewJWTService() *JWTService {
	return &JWTService{}
}

// GenerateToken tạo JWT token
func (j *JWTService) GenerateToken(user *entities.User) (string, error) {
	claims := JWTClaims{
		UserID: user.UserID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24h
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "nxrGO",
			Subject:   string(rune(user.UserID)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken xác thực và parse JWT token
func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.NewError(errors.INTERNAL_ERROR, "Unexpected signing method", http.StatusInternalServerError, nil)
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, errors.NewError(errors.INTERNAL_ERROR, "Unexpected error", http.StatusInternalServerError, nil)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.NewError(errors.INTERNAL_ERROR, "Invalid token", http.StatusInternalServerError, nil)

}

// GenerateRefreshToken tạo refresh token
func (j *JWTService) GenerateRefreshToken(user *entities.User) (string, error) {
	claims := JWTClaims{
		UserID: user.UserID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "nxrGO",
			Subject:   string(rune(user.UserID)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
