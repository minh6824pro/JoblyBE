package middleware

import (
	entities "Jobly/internal/entities"
	"Jobly/internal/errors"
	"Jobly/internal/jwt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type AuthMiddleware struct {
	jwtService *jwt.JWTService
}

func NewAuthMiddleware(jwtService *jwt.JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

// RequireAuth middleware xác thực JWT
func (a *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.WriteError(c,
				errors.NewError(
					errors.UNAUTHORIZED,
					"Authorization header required",
					http.StatusUnauthorized,
					nil))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			errors.WriteError(c,
				errors.NewError(
					errors.UNAUTHORIZED,
					"Bearer token required",
					http.StatusUnauthorized,
					nil))
			c.Abort()
			return
		}

		claims, err := a.jwtService.ValidateToken(tokenString)
		if err != nil {
			errors.WriteError(c,
				errors.NewError(
					errors.UNAUTHORIZED,
					"Invalid token",
					http.StatusUnauthorized,
					nil))
			c.Abort()
			return
		}

		// Lưu thông tin user vào context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// RequireRole middleware kiểm tra quyền
func (a *AuthMiddleware) RequireRole(role entities.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			errors.WriteError(c,
				errors.NewError(
					errors.UNAUTHORIZED,
					"Unauthorized",
					http.StatusUnauthorized,
					nil))
			c.Abort()
			return
		}

		if userRole != role {
			errors.WriteError(c,
				errors.NewError(
					errors.FORBIDDEN,
					"Insufficient permissions",
					http.StatusForbidden,
					nil))
			c.Abort()
			return
		}

		c.Next()
	}
}

func (a *AuthMiddleware) GetAuthIfExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Không có header → bỏ qua, vẫn cho request đi tiếp
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			// Sai format Bearer → bỏ qua (không abort)
			c.Next()
			return
		}

		claims, err := a.jwtService.ValidateToken(tokenString)
		if err != nil {
			// Token không hợp lệ → bỏ qua
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}
