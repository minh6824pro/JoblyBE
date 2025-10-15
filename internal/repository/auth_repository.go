package repository

import (
	entities "Jobly/internal/entities"
	"gorm.io/gorm"
)

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

type AuthRepository interface {
	FindByEmail(email string) (*entities.User, error)
	FindByID(id uint) (*entities.User, error)
	Create(user *entities.User) error
	Update(user *entities.User) error
	IsEmailExists(email string) bool
}

func (r *authRepository) FindByEmail(email string) (*entities.User, error) {
	var user entities.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) FindByID(id uint) (*entities.User, error) {
	var user entities.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) Create(user *entities.User) error {
	return r.db.Create(user).Error
}

func (r *authRepository) Update(user *entities.User) error {
	return r.db.Save(user).Error
}

func (r *authRepository) IsEmailExists(email string) bool {
	var count int64
	r.db.Model(&entities.User{}).Where("email = ?", email).Count(&count)
	return count > 0
}
