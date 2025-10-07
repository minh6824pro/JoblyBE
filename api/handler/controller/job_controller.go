package controller

import (
	"Jobly/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type JobController struct {
	jobService service.JobService
}

func NewJobController(jobService service.JobService) *JobController {
	return &JobController{
		jobService,
	}
}

func (c *JobController) GetJobList(ctx *gin.Context) {
	// Get page parameter from query string, default to 1
	pageStr := ctx.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	// Get keywords from query string (can be multiple)
	// Example: ?keywords=golang&keywords=backend or ?keywords=golang,backend
	keywords := ctx.QueryArray("keywords")

	// Call service to get job list
	jobs, err := c.jobService.ListJob(ctx.Request.Context(), page, keywords)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch jobs",
		})
		return
	}

	// Return success response
	ctx.JSON(http.StatusOK, jobs)
}
