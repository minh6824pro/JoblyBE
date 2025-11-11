package biz

import (
	"context"
	"errors"

	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive = errors.New("user account is inactive")
	ErrWeakPassword = errors.New("password is too weak")
	ErrInvalidEmail = errors.New("invalid email format")
	ErrInvalidPhone = errors.New("invalid phone format")
)

// User entity in business layer
type User struct {
	UserID      string
	FullName    string
	Email       string
	Password    string // hashed password
	PhoneNumber string
	Role        Role
	Active      bool
	LastLogin   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UserRepo interface định nghĩa các methods để tương tác với database
type UserRepo interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
	UpdateUser(ctx context.Context, user *User) error
}

// AuthUseCase handles authentication business logic
type AuthUseCase struct {
	userRepo UserRepo
	log      *log.Helper
}

// NewAuthUsecase creates a new AuthUseCase
func NewAuthUsecase(userRepo UserRepo, logger log.Logger) *AuthUseCase {
	return &AuthUseCase{
		userRepo: userRepo,
		log:      log.NewHelper(logger),
	}
}

// RegisterUser registers a new user
func (uc *AuthUseCase) RegisterUser(ctx context.Context, fullName, email, password, phoneNumber string) (*User, error) {
	uc.log.WithContext(ctx).Infof("RegisterUser: %s", email)

	// Check if user already exists
	existingUser, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		uc.log.Errorf("failed to hash password: %v", err)
		return nil, err
	}

	// Create user
	user := &User{
		FullName:    fullName,
		Email:       email,
		Password:    string(hashedPassword),
		PhoneNumber: phoneNumber,
		Role:        RoleUser,
		Active:      true,
	}

	createdUser, err := uc.userRepo.CreateUser(ctx, user)
	if err != nil {
		uc.log.Errorf("failed to create user: %v", err)
		return nil, err
	}

	return createdUser, nil
}

// Login authenticates a user
func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*User, error) {
	uc.log.WithContext(ctx).Infof("Login: %s", email)

	// Get user by email
	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.Active {
		return nil, ErrUserInactive
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	_ = uc.userRepo.UpdateLastLogin(ctx, user.UserID)

	return user, nil
}

// GetUserByID retrieves user by ID
func (uc *AuthUseCase) GetUserByID(ctx context.Context, userID string) (*User, error) {
	uc.log.WithContext(ctx).Infof("GetUserByID: %s", userID)

	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// ValidateEmail validates email format
func (uc *AuthUseCase) ValidateEmail(email string) bool {
	// Simple validation - you can use a regex for more robust validation
	return len(email) > 0 && len(email) < 255
}

// ValidatePassword validates password strength
func (uc *AuthUseCase) ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	// Add more password validation rules as needed
	return nil
}

// UpdateProfile updates user profile information
func (uc *AuthUseCase) UpdateProfile(ctx context.Context, userID, fullName, phoneNumber string) (*User, error) {
	uc.log.WithContext(ctx).Infof("UpdateProfile: %s", userID)

	// Get existing user
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Update fields if provided
	if fullName != "" {
		user.FullName = fullName
	}
	if phoneNumber != "" {
		user.PhoneNumber = phoneNumber
	}

	// Update user in database
	err = uc.userRepo.UpdateUser(ctx, user)
	if err != nil {
		uc.log.Errorf("failed to update user: %v", err)
		return nil, err
	}

	return user, nil
}

// ChangePassword changes user password
func (uc *AuthUseCase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	uc.log.WithContext(ctx).Infof("ChangePassword: %s", userID)

	// Get user
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Verify old password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
	if err != nil {
		return ErrInvalidCredentials
	}

	// Validate new password strength
	if err := uc.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		uc.log.Errorf("failed to hash password: %v", err)
		return err
	}

	// Update password
	user.Password = string(hashedPassword)
	err = uc.userRepo.UpdateUser(ctx, user)
	if err != nil {
		uc.log.Errorf("failed to update password: %v", err)
		return err
	}

	return nil
}
