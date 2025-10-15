package controller

import (
	"Jobly/internal/dto"
	customErr "Jobly/internal/errors"
	"Jobly/internal/service"
	"Jobly/internal/util"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

type AuthController struct {
	authService service.AuthService
}

func NewAuthController(authService service.AuthService) *AuthController {
	return &AuthController{authService}
}

// Register godoc
// @Summary User registration
// @Description Register a new user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param registerRequest body dto.RegisterRequest true "Register request payload"
// @Success 201 {object} dto.AuthResponse "Successfully registered"
// @Router /auth/register [post]
func (a *AuthController) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if errors.Is(err, io.EOF) {
			customErr.WriteError(c, customErr.NewError(
				customErr.BAD_REQUEST,
				"Request body is empty",
				http.StatusBadRequest,
				err,
			))
			return
		}

		if util.HandleValidationError(c, err) {
			return
		}
		customErr.WriteError(c, err)
		return
	}

	resp, err := a.authService.Register(req)
	if err != nil {
		customErr.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login godoc
// @Summary User Login
// @Description Login to get access token
// @Tags auth
// @Accept json
// @Produce json
// @Param registerRequest body dto.LoginRequest true "Register request payload"
// @Success 200 {object} dto.AuthResponse "Successfully registered"
// @Router /auth/login [post]
func (a *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if errors.Is(err, io.EOF) {
			customErr.WriteError(c, customErr.NewError(
				customErr.BAD_REQUEST,
				"Request body is empty",
				http.StatusBadRequest,
				err,
			))
			return
		}
		if util.HandleValidationError(c, err) {
			return
		}
		customErr.WriteError(c, err)
		return
	}

	resp, err := a.authService.Login(req)
	if err != nil {
		customErr.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetProfile godoc
// @Summary Get current profile
// @Description Get current profile
// @Tags auth
// @Accept json
// @Produce json
// @Security     BearerAuth
// @Success 200 {object} models.User
// @Router /user/profile [get]
func (a *AuthController) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		customErr.WriteError(c, customErr.NewError(
			customErr.UNAUTHORIZED,
			"Unauthorized",
			http.StatusUnauthorized,
			nil))
		return
	}

	profile, err := a.authService.GetProfile(userID.(uint))
	if err != nil {
		customErr.WriteError(c, customErr.NewError(
			customErr.UNAUTHORIZED,
			"User not found",
			http.StatusUnauthorized,
			nil))
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": profile})
}

// RefreshToken godoc
// @Summary Refresh Token
// @Description Refresh access token with body "refresh_token"
// @Tags auth
// @Accept json
// @Produce json
// @Param refreshToken body object true "Refresh token payload"
// @Router /auth/refresh [post]
func (a *AuthController) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newAccessToken, err := a.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken})
}
