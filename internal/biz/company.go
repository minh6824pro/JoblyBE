package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrCompanyNotFound      = errors.New("company not found")
	ErrCompanyAlreadyExists = errors.New("company already exists")
	ErrInvalidCompanyData   = errors.New("invalid company data")
)

// Company entity
type Company struct {
	ID          string
	Name        string
	Description string
	Website     string
	LogoURL     string
	Industry    string
	CompanySize string
	Location    string
	FoundedYear string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CompanyRepo interface
type CompanyRepo interface {
	CreateCompany(ctx context.Context, company *Company) (*Company, error)
	UpdateCompany(ctx context.Context, company *Company) error
	DeleteCompany(ctx context.Context, id string) error
	GetCompany(ctx context.Context, id string) (*Company, error)
	GetCompanyByName(ctx context.Context, name string) (*Company, error)
	ListCompanies(ctx context.Context, filter *CompanyFilter, page, pageSize int32) ([]*Company, int32, error)
}

// CompanyFilter for filtering and searching companies
type CompanyFilter struct {
	Industry string
	Location string
	Keyword  string
}

// CompanyUseCase handles company business logic
type CompanyUseCase struct {
	companyRepo CompanyRepo
	log         *log.Helper
}

// NewCompanyUseCase creates a new company use case
func NewCompanyUseCase(companyRepo CompanyRepo, logger log.Logger) *CompanyUseCase {
	return &CompanyUseCase{
		companyRepo: companyRepo,
		log:         log.NewHelper(logger),
	}
}

// CreateCompany creates a new company
func (uc *CompanyUseCase) CreateCompany(ctx context.Context, company *Company) (*Company, error) {

	// Check if company already exists
	existingCompany, err := uc.companyRepo.GetCompanyByName(ctx, company.Name)
	if err != nil {
		return nil, err
	}
	if existingCompany != nil {
		return nil, ErrCompanyAlreadyExists
	}

	// Validate company data
	if err := uc.validateCompany(company); err != nil {
		return nil, err
	}

	// Create company
	createdCompany, err := uc.companyRepo.CreateCompany(ctx, company)
	if err != nil {
		return nil, err
	}

	return createdCompany, nil
}

// UpdateCompany updates an existing company
func (uc *CompanyUseCase) UpdateCompany(ctx context.Context, company *Company) (*Company, error) {

	// Get existing company
	existingCompany, err := uc.companyRepo.GetCompany(ctx, company.ID)
	if err != nil {
		return nil, err
	}
	if existingCompany == nil {
		return nil, ErrCompanyNotFound
	}

	// Validate company data
	if err := uc.validateCompany(company); err != nil {
		return nil, err
	}

	// Update company
	if err := uc.companyRepo.UpdateCompany(ctx, company); err != nil {
		return nil, err
	}

	// Get updated company
	updatedCompany, err := uc.companyRepo.GetCompany(ctx, company.ID)
	if err != nil {
		return nil, err
	}

	return updatedCompany, nil
}

// DeleteCompany deletes a company
func (uc *CompanyUseCase) DeleteCompany(ctx context.Context, id string) error {

	// Get existing company
	existingCompany, err := uc.companyRepo.GetCompany(ctx, id)
	if err != nil {
		return err
	}
	if existingCompany == nil {
		return ErrCompanyNotFound
	}

	// Delete company
	if err := uc.companyRepo.DeleteCompany(ctx, id); err != nil {
		return err
	}

	return nil
}

// GetCompany retrieves a company by ID
func (uc *CompanyUseCase) GetCompany(ctx context.Context, id string) (*Company, error) {

	company, err := uc.companyRepo.GetCompany(ctx, id)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, ErrCompanyNotFound
	}

	return company, nil
}

// ListCompanies lists companies with filters and pagination
func (uc *CompanyUseCase) ListCompanies(ctx context.Context, filter *CompanyFilter, page, pageSize int32) ([]*Company, int32, error) {

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	companies, total, err := uc.companyRepo.ListCompanies(ctx, filter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return companies, total, nil
}

// validateCompany validates company data
func (uc *CompanyUseCase) validateCompany(company *Company) error {
	if company.Name == "" {
		return ErrInvalidCompanyData
	}

	return nil
}
