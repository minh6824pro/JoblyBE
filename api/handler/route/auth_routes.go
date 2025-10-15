package route

import (
	entities "Jobly/internal/entities"
	"Jobly/internal/modules"
	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(rg *gin.RouterGroup, authModule *modules.AuthModule) {

	auth := rg.Group("/auth")
	{
		auth.POST("/register", authModule.AuthController.Register)
		auth.POST("/login", authModule.AuthController.Login)
		auth.POST("/refresh", authModule.AuthController.RefreshToken)
	}

	// Protected routes
	protected := rg.Group("/user")
	protected.Use(authModule.AuthMiddleware.RequireAuth())
	{
		protected.GET("/profile", authModule.AuthController.GetProfile)

	}

	// Admin only routes
	admin := rg.Group("/admin")
	admin.Use(authModule.AuthMiddleware.RequireAuth())
	admin.Use(authModule.AuthMiddleware.RequireRole(entities.RoleAdmin))
	{
		// Thêm các routes admin ở đây
	}
}
