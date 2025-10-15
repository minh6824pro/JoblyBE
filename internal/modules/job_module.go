package modules

import (
	"Jobly/api/handler/controller"
	"Jobly/api/middleware"
)

type JobModule struct {
	JobController  *controller.JobController
	AuthMiddleware *middleware.AuthMiddleware
}
