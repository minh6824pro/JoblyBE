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

	// Default field tag to "summary" if not provided
	fieldTag := req.FieldTag
	if fieldTag == "" {
		fieldTag = "summary"
	}

	// Convert proto to biz ResumeDetail
	resumeDetail := s.protoToResumeDetail(req.ResumeDetail)

	content, err := s.uc.GenerateCVDescription(ctx, resumeDetail, claims.UserID, fieldTag, req.CurrentInput)
	if err != nil {
		return nil, err
	}

	return &pb.GenerateCVDescriptionReply{
		Content: content,
	}, nil
}

func (s *ResumeService) UpdateCVEditStatus(ctx context.Context, req *pb.UpdateCVEditStatusRequest) (*pb.UpdateCVEditStatusReply, error) {
	// Get user ID from JWT claims
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Update CV edit status
	if err := s.uc.UpdateCVEditStatus(ctx, req.ResumeId, req.EditId, req.Status, claims.UserID); err != nil {
		return nil, err
	}

	return &pb.UpdateCVEditStatusReply{
		Success: true,
		Message: "CV edit status updated successfully",
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
		Projects:       s.projectArrayToPb(detail.Projects),
		Certifications: detail.Certifications,
		Languages:      detail.Languages,
		Achievements:   detail.Achievements,
		Evaluations:    s.evaluationsArrayToPb(detail.Evaluations),
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
				Description:    edu.Description,
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
				Description:      exp.Description,
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
		Projects:       s.protoToProjectArray(detail.Projects),
		Certifications: detail.Certifications,
		Languages:      detail.Languages,
		Achievements:   detail.Achievements,
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
				Description:    edu.Description,
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
				Description:      exp.Description,
				Responsibilities: exp.Responsibilities,
				Achievements:     exp.Achievements,
			})
		}
	}
	return result
}

func (s *ResumeService) projectArrayToPb(projectList []*biz.Project) []*pb.Project {
	if projectList == nil {
		return nil
	}
	result := make([]*pb.Project, 0, len(projectList))
	for _, proj := range projectList {
		if proj != nil {
			result = append(result, &pb.Project{
				Name:         proj.Name,
				Description:  proj.Description,
				Technologies: proj.Technologies,
				Url:          proj.Url,
				Duration:     proj.Duration,
				Role:         proj.Role,
				Achievements: proj.Achievements,
			})
		}
	}
	return result
}

func (s *ResumeService) protoToProjectArray(projectList []*pb.Project) []*biz.Project {
	if projectList == nil {
		return nil
	}
	result := make([]*biz.Project, 0, len(projectList))
	for _, proj := range projectList {
		if proj != nil {
			result = append(result, &biz.Project{
				Name:         proj.Name,
				Description:  proj.Description,
				Technologies: proj.Technologies,
				Url:          proj.Url,
				Duration:     proj.Duration,
				Role:         proj.Role,
				Achievements: proj.Achievements,
			})
		}
	}
	return result
}

func (s *ResumeService) evaluationsArrayToPb(evalList []*biz.ResumeEvaluation) []*pb.ResumeEvaluation {
	if evalList == nil {
		return nil
	}
	result := make([]*pb.ResumeEvaluation, 0, len(evalList))
	for _, eval := range evalList {
		if eval != nil {
			pbEval := &pb.ResumeEvaluation{
				CvName:          eval.CVName,
				OverallScore:    eval.OverallScore,
				Grade:           eval.Grade,
				Strengths:       eval.Strengths,
				Weaknesses:      eval.Weaknesses,
				Recommendations: eval.Recommendations,
				JobsAnalyzed:    int32(eval.JobsAnalyzed),
				EvaluatedAt:     eval.EvaluatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}

			// Convert ScoreBreakdown
			if eval.ScoreBreakdown != nil {
				pbEval.ScoreBreakdown = &pb.ResumeScoreBreakdown{
					SkillsScore:       eval.ScoreBreakdown.SkillsScore,
					ExperienceScore:   eval.ScoreBreakdown.ExperienceScore,
					EducationScore:    eval.ScoreBreakdown.EducationScore,
					CompletenessScore: eval.ScoreBreakdown.CompletenessScore,
					JobAlignmentScore: eval.ScoreBreakdown.JobAlignmentScore,
					PresentationScore: eval.ScoreBreakdown.PresentationScore,
				}
			}

			// Convert CVEdits
			if len(eval.CVEdits) > 0 {
				pbEval.CvEdits = make([]*pb.ResumeCVEdit, 0, len(eval.CVEdits))
				for _, edit := range eval.CVEdits {
					if edit != nil {
						pbEval.CvEdits = append(pbEval.CvEdits, &pb.ResumeCVEdit{
							Id:             edit.ID,
							FieldPath:      edit.FieldPath,
							Action:         edit.Action,
							CurrentValue:   edit.CurrentValue,
							SuggestedValue: edit.SuggestedValue,
							Reason:         edit.Reason,
							Priority:       edit.Priority,
							ImpactScore:    edit.ImpactScore,
							Status:         edit.Status,
						})
					}
				}
			}

			result = append(result, pbEval)
		}
	}
	return result
}
