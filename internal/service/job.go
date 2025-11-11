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
	// Convert proto to biz
	job := &biz.JobPosting{
		CompanyID:             req.CompanyId,
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
	}

	created, err := s.jobPostingUseCase.CreateJobPosting(ctx, job)
	if err != nil {
		return nil, err
	}

	return s.jobToPb(created), nil
}

func (s *JobPostingService) UpdateJobPosting(ctx context.Context, req *pb.UpdateJobPostingRequest) (*pb.JobPostingReply, error) {
	job := &biz.JobPosting{
		ID:                    req.Id,
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
		CompanyID: req.CompanyId,
		Location:  req.Location,
		JobType:   biz.JobType(req.JobType),
		Level:     biz.Level(req.Level),
		Keyword:   req.Keyword,
		JobTech:   req.JobTech,
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
