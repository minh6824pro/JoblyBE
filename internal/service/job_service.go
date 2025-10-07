package service

import (
	"Jobly/api/handler/controller/httpresponse"
	"Jobly/internal/repository"
	"context"
)

type JobServiceImpl struct {
	jobRepo repository.JobRepository
}

type JobService interface {
	ListJob(ctx context.Context, page int, keywords []string) (httpresponse.ListJobResponse, error)
}

func NewJobServiceImpl(jobRepo repository.JobRepository) JobService {
	return &JobServiceImpl{
		jobRepo: jobRepo,
	}
}

func (s *JobServiceImpl) ListJob(ctx context.Context, page int, keywords []string) (httpresponse.ListJobResponse, error) {
	return s.jobRepo.GetJobList(ctx, page, keywords)
}
