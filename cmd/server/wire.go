//go:build wireinject
// +build wireinject

package main

import (
	"Jobly/api/handler/controller"
	"Jobly/internal/repository"
	"Jobly/internal/service"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitJobModule(db *gorm.DB) *controller.JobController {
	wire.Build(
		repository.NewJobGormRepository,
		service.NewJobServiceImpl,
		controller.NewJobController,
	)
	return &controller.JobController{}
}
