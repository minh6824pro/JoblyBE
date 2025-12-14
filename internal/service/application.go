package service

import (
	pb "JobblyBE/api/application/v1"
	"JobblyBE/internal/biz"
	"JobblyBE/pkg/middleware/auth"
	"context"
)

type JobApplicationService struct {
	pb.UnimplementedJobApplicationServer
	uc *biz.JobApplicationUseCase
}

func NewJobApplicationService(uc *biz.JobApplicationUseCase) *JobApplicationService {
	return &JobApplicationService{uc: uc}
}

// ApplyJob handles job application creation
func (s *JobApplicationService) ApplyJob(ctx context.Context, req *pb.ApplyJobRequest) (*pb.JobApplicationReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Create application
	app, err := s.uc.ApplyJob(ctx, claims.UserID, req.JobId, req.ResumeId)
	if err != nil {
		return nil, err
	}

	return &pb.JobApplicationReply{
		Application: s.applicationToInfoPb(app),
	}, nil
}

// GetApplication retrieves application detail with full resume
func (s *JobApplicationService) GetApplication(ctx context.Context, req *pb.GetApplicationRequest) (*pb.JobApplicationDetailReply, error) {
	app, err := s.uc.GetApplicationWithDetail(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.JobApplicationDetailReply{
		Application: s.applicationToDetailPb(app),
	}, nil
}

// ListUserApplications lists all applications for a user
func (s *JobApplicationService) ListUserApplications(ctx context.Context, req *pb.ListUserApplicationsRequest) (*pb.ListApplicationsReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	apps, total, err := s.uc.ListUserApplications(ctx, claims.UserID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	pbApps := make([]*pb.JobApplicationInfo, len(apps))
	for i, app := range apps {
		pbApps[i] = s.applicationToInfoPb(app)
	}

	return &pb.ListApplicationsReply{
		Applications: pbApps,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}, nil
}

// ListJobApplications lists all applications for a job posting (for HR)
func (s *JobApplicationService) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListApplicationsReply, error) {
	// TODO: Add authorization check - only HR/company owner can view job applications

	status := s.protoStatusToBizStatus(req.Status)
	apps, total, err := s.uc.ListJobApplications(ctx, req.JobId, status, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	pbApps := make([]*pb.JobApplicationInfo, len(apps))
	for i, app := range apps {
		pbApps[i] = s.applicationToInfoPb(app)
	}

	return &pb.ListApplicationsReply{
		Applications: pbApps,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}, nil
}

// UpdateApplicationStatus updates application status (for HR)
func (s *JobApplicationService) UpdateApplicationStatus(ctx context.Context, req *pb.UpdateApplicationStatusRequest) (*pb.JobApplicationReply, error) {
	// TODO: Add authorization check - only HR/company owner can update status

	status := s.protoStatusToBizStatus(req.Status)
	err := s.uc.UpdateApplicationStatus(ctx, req.Id, status, req.HrNote)
	if err != nil {
		return nil, err
	}

	// Get updated application
	app, err := s.uc.GetApplication(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.JobApplicationReply{
		Application: s.applicationToInfoPb(app),
	}, nil
}

// WithdrawApplication withdraws an application (for user)
func (s *JobApplicationService) WithdrawApplication(ctx context.Context, req *pb.WithdrawApplicationRequest) (*pb.WithdrawApplicationReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	err = s.uc.WithdrawApplication(ctx, req.Id, claims.UserID)
	if err != nil {
		return &pb.WithdrawApplicationReply{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.WithdrawApplicationReply{
		Success: true,
		Message: "Application withdrawn successfully",
	}, nil
}

// Helper functions to convert between proto and biz types

func (s *JobApplicationService) applicationToInfoPb(app *biz.JobApplication) *pb.JobApplicationInfo {
	info := &pb.JobApplicationInfo{
		Id:            app.ID,
		UserId:        app.UserID,
		ResumeId:      app.ResumeID,
		JobId:         app.JobID,
		ApplicantName: app.ApplicantName,
		University:    app.University,
		Status:        s.bizStatusToProtoStatus(app.Status),
		AppliedAt:     app.AppliedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     app.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		HrNote:        app.HRNote,
	}

	// Add job info if available
	if app.JobInfo != nil {
		info.JobInfo = &pb.JobInfo{
			Id:          app.JobInfo.ID,
			Title:       app.JobInfo.Title,
			CompanyName: app.JobInfo.CompanyName,
			Location:    app.JobInfo.Location,
		}
	}

	return info
}

func (s *JobApplicationService) applicationToDetailPb(app *biz.JobApplication) *pb.JobApplicationDetail {
	detail := &pb.JobApplicationDetail{
		Id:            app.ID,
		UserId:        app.UserID,
		ResumeId:      app.ResumeID,
		JobId:         app.JobID,
		ApplicantName: app.ApplicantName,
		University:    app.University,
		Status:        s.bizStatusToProtoStatus(app.Status),
		AppliedAt:     app.AppliedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     app.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		HrNote:        app.HRNote,
	}

	// Add resume detail if available
	if app.ResumeDetail != nil {
		detail.ResumeDetail = s.resumeDetailToPb(app.ResumeDetail)
	}

	// Add job info if available
	if app.JobInfo != nil {
		detail.JobInfo = &pb.JobInfo{
			Id:          app.JobInfo.ID,
			Title:       app.JobInfo.Title,
			CompanyName: app.JobInfo.CompanyName,
			Location:    app.JobInfo.Location,
		}
	}

	return detail
}

func (s *JobApplicationService) resumeDetailToPb(detail *biz.ResumeDetail) *pb.ResumeDetail {
	pbEducation := make([]*pb.Education, len(detail.Education))
	for i, edu := range detail.Education {
		pbEducation[i] = &pb.Education{
			Degree:         edu.Degree,
			Institution:    edu.Institution,
			GraduationYear: edu.GraduationYear,
			Description:    edu.Description,
		}
	}

	pbExperience := make([]*pb.Experience, len(detail.Experience))
	for i, exp := range detail.Experience {
		pbExperience[i] = &pb.Experience{
			Title:            exp.Title,
			Company:          exp.Company,
			Duration:         exp.Duration,
			Responsibilities: exp.Responsibilities,
			Achievements:     exp.Achievements,
			Description:      exp.Description,
		}
	}

	pbProjects := make([]*pb.Project, len(detail.Projects))
	for i, proj := range detail.Projects {
		pbProjects[i] = &pb.Project{
			Name:         proj.Name,
			Description:  proj.Description,
			Technologies: proj.Technologies,
			Url:          proj.Url,
			Duration:     proj.Duration,
			Role:         proj.Role,
			Achievements: proj.Achievements,
		}
	}

	return &pb.ResumeDetail{
		Name:           detail.Name,
		Email:          detail.Email,
		Phone:          detail.Phone,
		Summary:        detail.Summary,
		Skills:         detail.Skills,
		Education:      pbEducation,
		Experience:     pbExperience,
		Projects:       pbProjects,
		Certifications: detail.Certifications,
		Languages:      detail.Languages,
	}
}

func (s *JobApplicationService) bizStatusToProtoStatus(status biz.ApplicationStatus) pb.ApplicationStatus {
	switch status {
	case biz.StatusApplied:
		return pb.ApplicationStatus_APPLIED
	case biz.StatusReviewing:
		return pb.ApplicationStatus_REVIEWING
	case biz.StatusAccepted:
		return pb.ApplicationStatus_ACCEPTED
	case biz.StatusRejected:
		return pb.ApplicationStatus_REJECTED
	default:
		return pb.ApplicationStatus_APPLIED
	}
}

func (s *JobApplicationService) protoStatusToBizStatus(status pb.ApplicationStatus) biz.ApplicationStatus {
	switch status {
	case pb.ApplicationStatus_APPLIED:
		return biz.StatusApplied
	case pb.ApplicationStatus_REVIEWING:
		return biz.StatusReviewing
	case pb.ApplicationStatus_ACCEPTED:
		return biz.StatusAccepted
	case pb.ApplicationStatus_REJECTED:
		return biz.StatusRejected
	default:
		return ""
	}
}
