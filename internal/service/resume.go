package service

import (
	pb "JobblyBE/api/resume/v1"
	"JobblyBE/internal/biz"
	"JobblyBE/pkg/middleware/auth"
	"context"
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

func (s *ResumeService) GenerateCVDescription(ctx context.Context, req *pb.GenerateCVDescriptionRequest) (*pb.GenerateCVDescriptionReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	description, err := s.uc.GenerateCVDescription(ctx, req.ResumeId, claims.UserID)
	if err != nil {
		return nil, err
	}

	return &pb.GenerateCVDescriptionReply{
		Description: description,
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
		Education:      s.educationArrayToPb(detail.Education),
		Experience:     s.experienceArrayToPb(detail.Experience),
		Certifications: detail.Certifications,
		Languages:      detail.Languages,
	}
}

func (s *ResumeService) educationArrayToPb(eduList []*biz.Education) []*pb.Education {
	if eduList == nil {
		return nil
	}
	result := make([]*pb.Education, 0, len(eduList))
	for _, edu := range eduList {
		if edu != nil {
			result = append(result, &pb.Education{
				Degree:         edu.Degree,
				Institution:    edu.Institution,
				GraduationYear: edu.GraduationYear,
			})
		}
	}
	return result
}

func (s *ResumeService) experienceArrayToPb(expList []*biz.Experience) []*pb.Experience {
	if expList == nil {
		return nil
	}
	result := make([]*pb.Experience, 0, len(expList))
	for _, exp := range expList {
		if exp != nil {
			result = append(result, &pb.Experience{
				Title:            exp.Title,
				Company:          exp.Company,
				Duration:         exp.Duration,
				Responsibilities: exp.Responsibilities,
				Achievements:     exp.Achievements,
			})
		}
	}
	return result
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
		Education:      s.protoToEducationArray(detail.Education),
		Experience:     s.protoToExperienceArray(detail.Experience),
		Certifications: detail.Certifications,
		Languages:      detail.Languages,
	}
}

func (s *ResumeService) protoToEducationArray(eduList []*pb.Education) []*biz.Education {
	if eduList == nil {
		return nil
	}
	result := make([]*biz.Education, 0, len(eduList))
	for _, edu := range eduList {
		if edu != nil {
			result = append(result, &biz.Education{
				Degree:         edu.Degree,
				Institution:    edu.Institution,
				GraduationYear: edu.GraduationYear,
			})
		}
	}
	return result
}

func (s *ResumeService) protoToExperienceArray(expList []*pb.Experience) []*biz.Experience {
	if expList == nil {
		return nil
	}
	result := make([]*biz.Experience, 0, len(expList))
	for _, exp := range expList {
		if exp != nil {
			result = append(result, &biz.Experience{
				Title:            exp.Title,
				Company:          exp.Company,
				Duration:         exp.Duration,
				Responsibilities: exp.Responsibilities,
				Achievements:     exp.Achievements,
			})
		}
	}
	return result
}
