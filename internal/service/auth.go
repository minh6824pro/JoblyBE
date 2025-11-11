package service

import (
	"JobblyBE/internal/biz"
	"context"
	"errors"

	pb "JobblyBE/api/auth/v1"
	"JobblyBE/pkg/middleware/auth"

	"github.com/go-kratos/kratos/v2/log"
)

type AuthService struct {
	pb.UnimplementedAuthServer
	authUC    *biz.AuthUseCase
	jwtSecret string
	log       *log.Helper
}

func NewAuthService(authUC *biz.AuthUseCase, jwtSecret string, logger log.Logger) *AuthService {
	if jwtSecret == "" {
		panic("JWT secret cannot be empty")
	}
	
	logHelper := log.NewHelper(logger)
	logHelper.Infof("AuthService initialized with JWT secret length: %d", len(jwtSecret))
	
	return &AuthService{
		authUC:    authUC,
		jwtSecret: jwtSecret,
		log:       logHelper,
	}
}

func (s *AuthService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthReply, error) {
	s.log.WithContext(ctx).Infof("Register request for email: %s", req.Email)

	// Validate input
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return nil, pb.ErrorDataRequestInvalid("email, password and full_name are required")
	}

	// Validate email format
	if !s.authUC.ValidateEmail(req.Email) {
		return nil, pb.ErrorInvalidEmailFormat("invalid email format")
	}

	// Validate password strength
	if err := s.authUC.ValidatePassword(req.Password); err != nil {
		if errors.Is(err, biz.ErrWeakPassword) {
			return nil, pb.ErrorWeakPassword("password must be at least 8 characters")
		}
		return nil, pb.ErrorDataRequestInvalid("invalid password")
	}

	// Register user through use case
	user, err := s.authUC.RegisterUser(ctx, req.FullName, req.Email, req.Password, req.PhoneNumber)
	if err != nil {
		if errors.Is(err, biz.ErrUserAlreadyExists) {
			return nil, pb.ErrorEmailAlreadyExists("email already exists")
		}
		s.log.WithContext(ctx).Errorf("Failed to register user: %v", err)
		return nil, pb.ErrorSystemError("failed to register user")
	}

	// Generate token pair
	tokens, err := auth.NewTokenPair(
		user.UserID,
		user.Email,
		user.FullName,
		user.PhoneNumber,
		string(user.Role),
		s.jwtSecret,
	)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to generate tokens: %v", err)
		return nil, pb.ErrorSystemError("failed to generate authentication tokens")
	}

	return &pb.AuthReply{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: &pb.AuthReply_User{
			FullName:    user.FullName,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
			Role:        string(user.Role),
		},
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthReply, error) {
	s.log.WithContext(ctx).Infof("Login request for email: %s", req.Email)

	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, pb.ErrorDataRequestInvalid("email and password are required")
	}

	// Login through use case
	user, err := s.authUC.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidCredentials) {
			return nil, pb.ErrorInvalidCredentials("invalid email or password")
		}
		if errors.Is(err, biz.ErrUserInactive) {
			return nil, pb.ErrorUnauthorized("user account is inactive")
		}
		s.log.WithContext(ctx).Errorf("Login failed: %v", err)
		return nil, pb.ErrorSystemError("login failed")
	}

	// Generate token pair
	tokens, err := auth.NewTokenPair(
		user.UserID,
		user.Email,
		user.FullName,
		user.PhoneNumber,
		string(user.Role),
		s.jwtSecret,
	)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to generate tokens: %v", err)
		return nil, pb.ErrorSystemError("failed to generate authentication tokens")
	}

	return &pb.AuthReply{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: &pb.AuthReply_User{
			FullName:    user.FullName,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
			Role:        string(user.Role),
		},
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenReply, error) {
	s.log.WithContext(ctx).Info("RefreshToken request")

	// Validate input
	if req.RefreshToken == "" {
		return nil, pb.ErrorDataRequestInvalid("refresh_token is required")
	}

	// Validate refresh token và lấy user ID
	userID, err := auth.ValidateRefreshToken(req.RefreshToken, s.jwtSecret)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Invalid refresh token: %v", err)
		return nil, pb.ErrorJwtTokenInvalid("invalid refresh token: %v", err)
	}

	// Get user from database through use case
	user, err := s.authUC.GetUserByID(ctx, userID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("User not found: %v", err)
		return nil, pb.ErrorUserNotFound("user not found")
	}

	// Generate new access token
	accessToken, err := auth.GenerateAccessToken(
		user.UserID,
		user.Email,
		user.FullName,
		user.PhoneNumber,
		string(user.Role),
		s.jwtSecret,
		auth.AccessTokenDuration,
	)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to generate access token: %v", err)
		return nil, pb.ErrorSystemError("failed to generate access token")
	}

	return &pb.RefreshTokenReply{
		AccessToken: accessToken,
	}, nil
}

func (s *AuthService) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileReply, error) {
	s.log.WithContext(ctx).Info("GetProfile request")

	// Lấy claims từ context (được set bởi JWT middleware)
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to get claims from context: %v", err)
		return nil, pb.ErrorJwtTokenInvalid("failed to get user claims: %v", err)
	}

	// Get full user profile from database through use case
	user, err := s.authUC.GetUserByID(ctx, claims.UserID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("User not found: %v", err)
		return nil, pb.ErrorUserNotFound("user not found")
	}

	// Return user profile
	return &pb.GetProfileReply{
		FullName:    user.FullName,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		Role:        string(user.Role),
		// Password should NEVER be returned in response
	}, nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileReply, error) {
	s.log.WithContext(ctx).Info("UpdateProfile request")

	// Lấy claims từ context (được set bởi JWT middleware)
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to get claims from context: %v", err)
		return nil, pb.ErrorJwtTokenInvalid("failed to get user claims: %v", err)
	}

	// Validate input - at least one field must be provided
	if req.FullName == "" && req.PhoneNumber == "" {
		return nil, pb.ErrorDataRequestInvalid("at least one field (full_name or phone_number) must be provided")
	}

	// Update user profile through use case
	user, err := s.authUC.UpdateProfile(ctx, claims.UserID, req.FullName, req.PhoneNumber)
	if err != nil {
		if errors.Is(err, biz.ErrUserNotFound) {
			return nil, pb.ErrorUserNotFound("user not found")
		}
		s.log.WithContext(ctx).Errorf("Failed to update profile: %v", err)
		return nil, pb.ErrorSystemError("failed to update profile")
	}

	// Return updated profile
	return &pb.UpdateProfileReply{
		FullName:    user.FullName,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		Role:        string(user.Role),
	}, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordReply, error) {
	s.log.WithContext(ctx).Info("ChangePassword request")

	// Lấy claims từ context (được set bởi JWT middleware)
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to get claims from context: %v", err)
		return nil, pb.ErrorJwtTokenInvalid("failed to get user claims: %v", err)
	}

	// Validate input
	if req.OldPassword == "" || req.NewPassword == "" {
		return nil, pb.ErrorDataRequestInvalid("old_password and new_password are required")
	}

	// Check if old and new passwords are the same
	if req.OldPassword == req.NewPassword {
		return nil, pb.ErrorDataRequestInvalid("new password must be different from old password")
	}

	// Validate new password strength
	if err := s.authUC.ValidatePassword(req.NewPassword); err != nil {
		if errors.Is(err, biz.ErrWeakPassword) {
			return nil, pb.ErrorWeakPassword("new password must be at least 8 characters")
		}
		return nil, pb.ErrorDataRequestInvalid("invalid new password")
	}

	// Change password through use case
	err = s.authUC.ChangePassword(ctx, claims.UserID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidCredentials) {
			return nil, pb.ErrorInvalidCredentials("old password is incorrect")
		}
		if errors.Is(err, biz.ErrUserNotFound) {
			return nil, pb.ErrorUserNotFound("user not found")
		}
		if errors.Is(err, biz.ErrWeakPassword) {
			return nil, pb.ErrorWeakPassword("new password is too weak")
		}
		s.log.WithContext(ctx).Errorf("Failed to change password: %v", err)
		return nil, pb.ErrorSystemError("failed to change password")
	}

	return &pb.ChangePasswordReply{
		Message: "Password changed successfully",
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutReply, error) {
	s.log.WithContext(ctx).Info("Logout request")

	// Lấy claims từ context (được set bởi JWT middleware)
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Failed to get claims from context: %v", err)
		return nil, pb.ErrorJwtTokenInvalid("failed to get user claims: %v", err)
	}

	// Log the logout action
	s.log.WithContext(ctx).Infof("User %s logged out", claims.UserID)

	// TODO: Implement token blacklist with Redis
	// For now, we just return success
	// In production, you should:
	// 1. Add the access token to a blacklist in Redis with TTL = token expiration time
	// 2. Add the refresh token to the blacklist if provided
	// 3. Check blacklist in JWT middleware before validating token
	
	// Example implementation:
	// if req.RefreshToken != "" {
	//     s.authUC.BlacklistToken(ctx, req.RefreshToken)
	// }
	// s.authUC.BlacklistToken(ctx, currentAccessToken)

	return &pb.LogoutReply{
		Message: "Logout successful",
	}, nil
}
