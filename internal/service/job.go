package service

import (
	pb "JobblyBE/api/job/v1"
	"JobblyBE/internal/biz"
	"JobblyBE/pkg/middleware/auth"
	"context"
)

type JobPostingService struct {
	pb.UnimplementedJobPostingServer
	jobPostingUseCase   *biz.JobPostingUseCase
	userTrackingUseCase *biz.UserTrackingUseCase
}

func NewJobPostingService(jobPostingUsecase *biz.JobPostingUseCase, userTrackingUseCase *biz.UserTrackingUseCase) *JobPostingService {
	return &JobPostingService{jobPostingUseCase: jobPostingUsecase,
		userTrackingUseCase: userTrackingUseCase}
}

func (s *JobPostingService) CreateJobPosting(ctx context.Context, req *pb.CreateJobPostingRequest) (*pb.JobPostingReply, error) {
	// Get current user from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Use companyID from request or from claims
	companyID := req.CompanyId
	if companyID == "" {
		companyID = claims.CompanyID
	}

	// Convert proto to biz
	job := &biz.JobPosting{
		CompanyID:             companyID,
		Title:                 req.Title,
		Level:                 biz.Level(req.Level),
		JobType:               biz.JobType(req.JobType),
		SalaryMin:             req.SalaryMin,
		SalaryMax:             req.SalaryMax,
		SalaryCurrency:        req.SalaryCurrency,
		Location:              req.Location,
		ExperienceRequirement: req.ExperienceRequirement,
		Description:           req.Description,
		Responsibilities:      req.Responsibilities,
		Requirements:          req.Requirements,
		Benefits:              req.Benefits,
		JobTech:               req.JobTech,
		Active:                req.Active,
		CreatedBy:             claims.UserID, // Save user ID from token
	}

	created, err := s.jobPostingUseCase.CreateJobPosting(ctx, job)
	if err != nil {
		return nil, err
	}

	return s.jobToPb(created), nil
}

func (s *JobPostingService) UpdateJobPosting(ctx context.Context, req *pb.UpdateJobPostingRequest) (*pb.JobPostingReply, error) {
	// Get existing job
	existingJob, err := s.jobPostingUseCase.GetJobPosting(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	job := &biz.JobPosting{
		ID:                    req.Id,
		CompanyID:             existingJob.CompanyID, // Keep original companyID
		Title:                 req.Title,
		Level:                 biz.Level(req.Level),
		JobType:               biz.JobType(req.JobType),
		SalaryMin:             req.SalaryMin,
		SalaryMax:             req.SalaryMax,
		SalaryCurrency:        req.SalaryCurrency,
		Location:              req.Location,
		ExperienceRequirement: req.ExperienceRequirement,
		Description:           req.Description,
		Responsibilities:      req.Responsibilities,
		Requirements:          req.Requirements,
		Benefits:              req.Benefits,
		JobTech:               req.JobTech,
		Active:                req.Active,
	}

	updated, err := s.jobPostingUseCase.UpdateJobPosting(ctx, job)
	if err != nil {
		return nil, err
	}

	return s.jobToPb(updated), nil
}

func (s *JobPostingService) DeleteJobPosting(ctx context.Context, req *pb.DeleteJobPostingRequest) (*pb.DeleteJobPostingReply, error) {
	if err := s.jobPostingUseCase.DeleteJobPosting(ctx, req.Id); err != nil {
		return nil, err
	}

	return &pb.DeleteJobPostingReply{Message: "Job posting deleted successfully"}, nil
}

func (s *JobPostingService) GetJobPosting(ctx context.Context, req *pb.GetJobPostingRequest) (*pb.JobPostingReply, error) {
	job, err := s.jobPostingUseCase.GetJobPosting(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return s.jobToPb(job), nil
}

func (s *JobPostingService) ListJobPostings(ctx context.Context, req *pb.ListJobPostingsRequest) (*pb.ListJobPostingsReply, error) {

	filter := &biz.JobFilter{
		CompanyID:       req.CompanyId,
		Location:        req.Location,
		JobType:         biz.JobType(req.JobType),
		Level:           biz.Level(req.Level),
		Keyword:         req.Keyword,
		JobTech:         req.JobTech,
		IncludeInactive: req.IncludeInactive,
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err == nil {
		err = s.userTrackingUseCase.CreateUserTrackingJobFilter(ctx, claims.UserID, filter)
		if err != nil {
			return nil, err
		}
	}

	jobs, total, err := s.jobPostingUseCase.ListJobPostings(ctx, filter, int32(req.Page), int32(req.PageSize))
	if err != nil {
		return nil, err
	}

	results := make([]*pb.JobPostingReply, 0, len(jobs))
	for _, job := range jobs {
		results = append(results, s.jobToPb(job))
	}

	return &pb.ListJobPostingsReply{
		Jobs:     results,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// Helper function to convert biz.JobPosting to pb.JobPostingReply
func (s *JobPostingService) jobToPb(job *biz.JobPosting) *pb.JobPostingReply {
	reply := &pb.JobPostingReply{
		Id:                    job.ID,
		CompanyId:             job.CompanyID,
		Title:                 job.Title,
		Level:                 string(job.Level),
		JobType:               string(job.JobType),
		SalaryMin:             job.SalaryMin,
		SalaryMax:             job.SalaryMax,
		SalaryCurrency:        job.SalaryCurrency,
		Location:              job.Location,
		ExperienceRequirement: job.ExperienceRequirement,
		Description:           job.Description,
		Responsibilities:      job.Responsibilities,
		Requirements:          job.Requirements,
		Benefits:              job.Benefits,
		JobTech:               job.JobTech,
		Active:                job.Active,
		CreatedBy:             job.CreatedBy,
		CreatedAt:             job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if job.PostedAt != nil {
		reply.PostedAt = job.PostedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	if job.Company != nil {
		reply.Company = &pb.CompanyInfo{
			Id:          job.Company.ID,
			Name:        job.Company.Name,
			Description: job.Company.Description,
			Website:     job.Company.Website,
			LogoUrl:     job.Company.LogoURL,
			Industry:    job.Company.Industry,
			CompanySize: job.Company.CompanySize,
			Location:    job.Company.Location,
			FoundedYear: job.Company.FoundedYear,
		}
	}

	return reply
}

func (s *JobPostingService) ListMyJobs(ctx context.Context, req *pb.ListMyJobsRequest) (*pb.ListJobPostingsReply, error) {
	// Get current user from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Use companyID from claims or request
	companyID := claims.CompanyID
	if req.CompanyId != "" {
		companyID = req.CompanyId
	}

	if companyID == "" {
		return &pb.ListJobPostingsReply{
			Jobs:     []*pb.JobPostingReply{},
			Total:    0,
			Page:     req.Page,
			PageSize: req.PageSize,
		}, nil
	}

	// List all jobs for this company (uses companyID from JWT claims)
	jobs, total, err := s.jobPostingUseCase.ListMyJobs(ctx, claims.CompanyID, int32(req.Page), int32(req.PageSize), req.IncludeInactive)
	if err != nil {
		return nil, err
	}

	results := make([]*pb.JobPostingReply, 0, len(jobs))
	for _, job := range jobs {
		results = append(results, s.jobToPb(job))
	}

	return &pb.ListJobPostingsReply{
		Jobs:     results,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *JobPostingService) GetMyCreatedJobs(ctx context.Context, req *pb.GetMyCreatedJobsRequest) (*pb.ListJobPostingsReply, error) {
	// Get current user from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// List all jobs created by this user (uses userID from JWT claims)
	jobs, total, err := s.jobPostingUseCase.GetMyCreatedJobs(ctx, claims.UserID, int32(req.Page), int32(req.PageSize), req.IncludeInactive)
	if err != nil {
		return nil, err
	}

	results := make([]*pb.JobPostingReply, 0, len(jobs))
	for _, job := range jobs {
		results = append(results, s.jobToPb(job))
	}

	return &pb.ListJobPostingsReply{
		Jobs:     results,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
