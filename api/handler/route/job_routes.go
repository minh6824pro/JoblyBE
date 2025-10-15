package route

import (
	"Jobly/internal/modules"

	"github.com/gin-gonic/gin"
)

func RegisterJobRoutes(rg *gin.RouterGroup, jobModule *modules.JobModule) {
	jobs := rg.Group("/jobs")
	jobs.Use(jobModule.AuthMiddleware.GetAuthIfExists())
	{
		jobs.GET("", jobModule.JobController.GetJobList)
	}
}
