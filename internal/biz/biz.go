package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewAuthUsecase,
	NewJobPostingUseCase,
	NewCompanyUseCase,
	NewUserTrackingUseCase,
	NewResumeUseCase,
	NewUserUseCase,
	NewJobApplicationUseCase,
)

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)
