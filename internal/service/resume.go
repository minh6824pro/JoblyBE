package service

import (
	"context"

	pb "JobblyBE/api/resume/v1"
	"JobblyBE/internal/biz"
	"JobblyBE/pkg/middleware/auth"
)

type ResumeService struct {
	pb.UnimplementedResumeServer
	uc *biz.ResumeUseCase
}

func NewResumeService(uc *biz.ResumeUseCase) *ResumeService {
	return &ResumeService{uc: uc}
}

func (s *ResumeService) CreateResume(ctx context.Context, req *pb.CreateResumeRequest) (*pb.ResumeReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Convert proto to biz
	resume := &biz.Resume{
		UserID:       claims.UserID,
		ResumeDetail: s.protoToResumeDetail(req.ResumeDetail),
	}

	created, err := s.uc.CreateResume(ctx, resume)
	if err != nil {
		return nil, err
	}

	return s.resumeToPb(created), nil
}

func (s *ResumeService) UpdateResume(ctx context.Context, req *pb.UpdateResumeRequest) (*pb.ResumeReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	resume := &biz.Resume{
		ID:           req.Id,
		UserID:       claims.UserID,
		ResumeDetail: s.protoToResumeDetail(req.ResumeDetail),
	}

	updated, err := s.uc.UpdateResume(ctx, resume)
	if err != nil {
		return nil, err
	}

	return s.resumeToPb(updated), nil
}

func (s *ResumeService) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.ResumeReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	resume, err := s.uc.GetResume(ctx, req.Id, claims.UserID)
	if err != nil {
		return nil, err
	}

	return s.resumeToPb(resume), nil
}

func (s *ResumeService) ListResumes(ctx context.Context, req *pb.ListResumesRequest) (*pb.ListResumesReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	resumes, total, err := s.uc.ListResumes(ctx, claims.UserID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	results := make([]*pb.ResumeReply, 0, len(resumes))
	for _, resume := range resumes {
		results = append(results, s.resumeToPb(resume))
	}

	return &pb.ListResumesReply{
		Resumes:  results,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *ResumeService) DeleteResume(ctx context.Context, req *pb.DeleteResumeRequest) (*pb.DeleteResumeReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.uc.DeleteResume(ctx, req.Id, claims.UserID); err != nil {
		return nil, err
	}

	return &pb.DeleteResumeReply{
		Success: true,
		Message: "Resume deleted successfully",
	}, nil
}

// Helper functions to convert between proto and biz models
func (s *ResumeService) resumeToPb(resume *biz.Resume) *pb.ResumeReply {
	return &pb.ResumeReply{
		Id:           resume.ID,
		UserId:       resume.UserID,
		ResumeDetail: s.resumeDetailToPb(resume.ResumeDetail),
		Version:      resume.Version,
		CreatedAt:    resume.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *ResumeService) resumeDetailToPb(detail *biz.ResumeDetail) *pb.ResumeDetail {
	if detail == nil {
		return nil
	}

	return &pb.ResumeDetail{
		Name:           detail.Name,
		Email:          detail.Email,
		Phone:          detail.Phone,
		Summary:        detail.Summary,
		Skills:         detail.Skills,
		Education:      s.educationToPb(detail.Education),
		Experience:     s.experienceToPb(detail.Experience),
		Certifications: detail.Certifications,
		Languages:      detail.Languages,
	}
}

func (s *ResumeService) educationToPb(edu *biz.Education) *pb.Education {
	if edu == nil {
		return nil
	}

	return &pb.Education{
		Degree:         edu.Degree,
		Institution:    edu.Institution,
		GraduationYear: edu.GraduationYear,
	}
}

func (s *ResumeService) experienceToPb(exp *biz.Experience) *pb.Experience {
	if exp == nil {
		return nil
	}

	return &pb.Experience{
		Title:            exp.Title,
		Company:          exp.Company,
		Duration:         exp.Duration,
		Responsibilities: exp.Responsibilities,
		Achievements:     exp.Achievements,
	}
}

func (s *ResumeService) protoToResumeDetail(detail *pb.ResumeDetail) *biz.ResumeDetail {
	if detail == nil {
		return nil
	}

	return &biz.ResumeDetail{
		Name:           detail.Name,
		Email:          detail.Email,
		Phone:          detail.Phone,
		Summary:        detail.Summary,
		Skills:         detail.Skills,
		Education:      s.protoToEducation(detail.Education),
		Experience:     s.protoToExperience(detail.Experience),
		Certifications: detail.Certifications,
		Languages:      detail.Languages,
	}
}

func (s *ResumeService) protoToEducation(edu *pb.Education) *biz.Education {
	if edu == nil {
		return nil
	}

	return &biz.Education{
		Degree:         edu.Degree,
		Institution:    edu.Institution,
		GraduationYear: edu.GraduationYear,
	}
}

func (s *ResumeService) protoToExperience(exp *pb.Experience) *biz.Experience {
	if exp == nil {
		return nil
	}

	return &biz.Experience{
		Title:            exp.Title,
		Company:          exp.Company,
		Duration:         exp.Duration,
		Responsibilities: exp.Responsibilities,
		Achievements:     exp.Achievements,
	}
}
