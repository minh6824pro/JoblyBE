package route

import (
	"Jobly/api/handler/controller"

	"github.com/gin-gonic/gin"
)

func RegisterJobRoutes(rg *gin.RouterGroup, jobCtrl *controller.JobController) {
	jobs := rg.Group("/jobs")
	{
		jobs.GET("", jobCtrl.GetJobList)
	}
}
