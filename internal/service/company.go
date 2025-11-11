package service

import (
	"context"

	pb "JobblyBE/api/job/v1"
	"JobblyBE/internal/biz"
)

type CompanyService struct {
	pb.UnimplementedCompanyServer
	uc *biz.CompanyUseCase
}

func NewCompanyService(uc *biz.CompanyUseCase) *CompanyService {
	return &CompanyService{uc: uc}
}

func (s *CompanyService) CreateCompany(ctx context.Context, req *pb.CreateCompanyRequest) (*pb.CompanyReply, error) {
	company := &biz.Company{
		Name:        req.Name,
		Description: req.Description,
		Website:     req.Website,
		LogoURL:     req.LogoUrl,
		Industry:    req.Industry,
		CompanySize: req.CompanySize,
		Location:    req.Location,
		FoundedYear: req.FoundedYear,
	}

	created, err := s.uc.CreateCompany(ctx, company)
	if err != nil {
		return nil, err
	}

	return s.companyToPb(created), nil
}

func (s *CompanyService) UpdateCompany(ctx context.Context, req *pb.UpdateCompanyRequest) (*pb.CompanyReply, error) {
	company := &biz.Company{
		ID:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		Website:     req.Website,
		LogoURL:     req.LogoUrl,
		Industry:    req.Industry,
		CompanySize: req.CompanySize,
		Location:    req.Location,
		FoundedYear: req.FoundedYear,
	}

	updated, err := s.uc.UpdateCompany(ctx, company)
	if err != nil {
		return nil, err
	}

	return s.companyToPb(updated), nil
}

func (s *CompanyService) DeleteCompany(ctx context.Context, req *pb.DeleteCompanyRequest) (*pb.DeleteCompanyReply, error) {
	if err := s.uc.DeleteCompany(ctx, req.Id); err != nil {
		return nil, err
	}

	return &pb.DeleteCompanyReply{Success: true}, nil
}

func (s *CompanyService) GetCompany(ctx context.Context, req *pb.GetCompanyRequest) (*pb.CompanyReply, error) {
	company, err := s.uc.GetCompany(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return s.companyToPb(company), nil
}

func (s *CompanyService) ListCompanies(ctx context.Context, req *pb.ListCompaniesRequest) (*pb.ListCompaniesReply, error) {
	filter := &biz.CompanyFilter{
		Industry: req.Industry,
		Location: req.Location,
		Keyword:  req.Keyword,
	}

	companies, total, err := s.uc.ListCompanies(ctx, filter, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	results := make([]*pb.CompanyReply, 0, len(companies))
	for _, company := range companies {
		results = append(results, s.companyToPb(company))
	}

	return &pb.ListCompaniesReply{
		Companies: results,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}

// Helper function to convert biz.Company to pb.CompanyReply
func (s *CompanyService) companyToPb(company *biz.Company) *pb.CompanyReply {
	return &pb.CompanyReply{
		Id:          company.ID,
		Name:        company.Name,
		Description: company.Description,
		Website:     company.Website,
		LogoUrl:     company.LogoURL,
		Industry:    company.Industry,
		CompanySize: company.CompanySize,
		Location:    company.Location,
		FoundedYear: company.FoundedYear,
	}
}
