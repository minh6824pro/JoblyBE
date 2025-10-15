package repository

import (
	entities "Jobly/internal/entities"
	"context"
	"gorm.io/gorm"
)

type UserTrackingGormRepository struct {
	db *gorm.DB
}

type UserTrackingRepository interface {
	Create(ctx context.Context, userTracking entities.UserTracking) error
}

func NewUserTrackingRepository(db *gorm.DB) UserTrackingRepository {
	return UserTrackingGormRepository{
		db: db,
	}
}

func (r UserTrackingGormRepository) Create(ctx context.Context, userTracking entities.UserTracking) error {
	err := r.db.Create(&userTracking).Error
	if err != nil {
		return err
	}
	return nil
}
