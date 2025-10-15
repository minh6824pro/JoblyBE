//go:build wireinject
// +build wireinject

package main

import (
	"Jobly/api/handler/controller"
	"Jobly/api/middleware"
	"Jobly/internal/jwt"
	"Jobly/internal/modules"
	"Jobly/internal/repository"
	"Jobly/internal/service"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitJobModule(db *gorm.DB) *modules.JobModule {
	wire.Build(
		repository.NewJobRepository,
		service.NewJobService,
		jwt.NewJWTService,
		middleware.NewAuthMiddleware,
		repository.NewUserTrackingRepository,
		controller.NewJobController,
		wire.Struct(new(modules.JobModule), "*"))
	return nil
}

func InitAuthModule(db *gorm.DB) *modules.AuthModule {
	wire.Build(
		repository.NewAuthRepository,
		service.NewAuthService,
		jwt.NewJWTService,
		middleware.NewAuthMiddleware,
		controller.NewAuthController,
		wire.Struct(new(modules.AuthModule), "*"))
	return nil
}
