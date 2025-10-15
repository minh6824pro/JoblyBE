package modules

import (
	"Jobly/api/handler/controller"
	"Jobly/api/middleware"
)

type AuthModule struct {
	AuthController *controller.AuthController
	AuthMiddleware *middleware.AuthMiddleware
}
