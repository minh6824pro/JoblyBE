package service

import (
	"Jobly/internal/dto"
	entities "Jobly/internal/entities"
	"Jobly/internal/entities/tracking"
	"Jobly/internal/repository"
	"context"
	"encoding/json"
	"gorm.io/datatypes"
	"time"
)

type JobServiceImpl struct {
	jobRepo      repository.JobRepository
	trackingRepo repository.UserTrackingRepository
}

type JobService interface {
	ListJob(ctx context.Context, page int, keywords []string, userID uint) (dto.ListJobResponse, error)
}

func NewJobService(jobRepo repository.JobRepository, trackingRepo repository.UserTrackingRepository) JobService {
	return &JobServiceImpl{
		jobRepo:      jobRepo,
		trackingRepo: trackingRepo,
	}
}

func (s *JobServiceImpl) ListJob(ctx context.Context, page int, keywords []string, userID uint) (dto.ListJobResponse, error) {
	if userID != 0 {
		go func() {
			trackingSearch := tracking.TrackingSearch{
				Keyword:   keywords,
				Timestamp: time.Now().Unix(),
			}
			data, _ := json.Marshal(trackingSearch)

			userTracking := entities.UserTracking{
				UserID:       userID,
				TrackingType: entities.TrackingJobSearch,
				Metadata:     datatypes.JSON(data),
			}
			s.trackingRepo.Create(ctx, userTracking)
		}()
	}
	return s.jobRepo.GetJobList(ctx, page, keywords)
}
