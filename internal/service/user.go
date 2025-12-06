package service

import (
	"context"

	pb "JobblyBE/api/user/v1"
	"JobblyBE/internal/biz"
	"JobblyBE/pkg/middleware/auth"
)

type UserService struct {
	pb.UnimplementedUserServer
	uc       *biz.UserUseCase
	tracking *biz.UserTrackingUseCase
}

func NewUserService(uc *biz.UserUseCase, tracking *biz.UserTrackingUseCase) *UserService {
	return &UserService{
		uc:       uc,
		tracking: tracking,
	}
}

func (s *UserService) CreateTrackingJDTOS(ctx context.Context, req *pb.CreateTrackingJDTOSRequest) (*pb.CreateTrackingJDTOSReply, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.tracking.CreateUserTrackingJDTOS(ctx, claims.UserID, req.JobId, req.TimeOnSight); err != nil {
		return nil, err
	}
	return &pb.CreateTrackingJDTOSReply{}, nil
}

func (s *UserService) SaveJob(ctx context.Context, req *pb.SaveJobRequest) (*pb.SaveJobReply, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = s.uc.SaveJob(ctx, claims.UserID, req.JobId)
	if err != nil {
		return nil, err
	}

	return &pb.SaveJobReply{
		Success: true,
		Message: "Job saved successfully",
	}, nil
}

func (s *UserService) UnsaveJob(ctx context.Context, req *pb.UnsaveJobRequest) (*pb.UnsaveJobReply, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = s.uc.UnsaveJob(ctx, claims.UserID, req.JobId)
	if err != nil {
		return nil, err
	}

	return &pb.UnsaveJobReply{
		Success: true,
		Message: "Job unsaved successfully",
	}, nil
}

func (s *UserService) GetSavedJobs(ctx context.Context, req *pb.GetSavedJobsRequest) (*pb.GetSavedJobsReply, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Default pagination
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	jobs, total, err := s.uc.GetSavedJobs(ctx, claims.UserID, page, pageSize)
	if err != nil {
		return nil, err
	}

	// Convert to proto
	pbJobs := make([]*pb.JobPosting, 0, len(jobs))
	for _, job := range jobs {
		pbJobs = append(pbJobs, s.jobToPb(job))
	}

	return &pb.GetSavedJobsReply{
		Jobs:     pbJobs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Helper function to convert biz.JobPosting to pb.JobPosting
func (s *UserService) jobToPb(job *biz.JobPosting) *pb.JobPosting {
	var postedAt string
	if job.PostedAt != nil {
		postedAt = job.PostedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return &pb.JobPosting{
		Id:                    job.ID,
		CompanyId:             job.CompanyID,
		Title:                 job.Title,
		Level:                 string(job.Level),
		JobType:               string(job.JobType),
		SalaryMin:             job.SalaryMin,
		SalaryMax:             job.SalaryMax,
		SalaryCurrency:        job.SalaryCurrency,
		Location:              job.Location,
		PostedAt:              postedAt,
		ExperienceRequirement: job.ExperienceRequirement,
		Description:           job.Description,
		Responsibilities:      job.Responsibilities,
		Requirements:          job.Requirements,
		Benefits:              job.Benefits,
		JobTech:               job.JobTech,
		CreatedAt:             job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
