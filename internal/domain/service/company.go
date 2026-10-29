package service

import (
	"company-registry/internal/domain/entity"
	"company-registry/internal/domain/errs"
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"

	"company-registry/internal/domain/enum"
)

type Service struct {
	companySource CompanySource
}

func NewService(
	companySource CompanySource,
) *Service {
	return &Service{
		companySource: companySource,
	}
}

type CompanySource interface {
	GetCompany(ctx context.Context, id uuid.UUID) (*entity.Company, error)
	CreateCompany(ctx context.Context, company *entity.Company) error
	UpdateCompany(ctx context.Context, company *entity.Company) error
	DeleteCompany(ctx context.Context, id uuid.UUID) error
}

func (s *Service) CreateCompany(ctx context.Context, company *entity.Company) (uuid.UUID, error) {
	company.ID = uuid.New()

	if err := s.ValidateCompany(company); err != nil {
		return uuid.UUID{}, fmt.Errorf("ValidateCompany: %w", err)
	}

	if err := s.companySource.CreateCompany(ctx, company); err != nil {
		return uuid.UUID{}, fmt.Errorf("companySource.CreateCompany: %w", err)
	}

	return company.ID, nil
}

func (s *Service) GetCompany(ctx context.Context, id uuid.UUID) (*entity.Company, error) {
	return s.companySource.GetCompany(ctx, id)
}

func (s *Service) PatchCompany(ctx context.Context, id uuid.UUID, patch entity.CompanyPatch) (*entity.Company, error) {
	company, err := s.companySource.GetCompany(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("companySource.GetCompany: %w", err)
	}

	company.Apply(patch)

	if err := s.ValidateCompany(company); err != nil {
		return nil, err
	}

	if err := s.companySource.UpdateCompany(ctx, company); err != nil {
		return nil, fmt.Errorf("companySource.UpdateCompany: %w", err)
	}

	return company, nil
}

func (s *Service) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	return s.companySource.DeleteCompany(ctx, id)
}

func (s *Service) ValidateCompany(company *entity.Company) error {
	if company.Name == "" {
		return fmt.Errorf("%w: name is required", errs.ErrValidation)
	}
	if utf8.RuneCountInString(company.Name) > 15 {
		return fmt.Errorf("%w: name must be at most 15 characters", errs.ErrValidation)
	}
	if company.Description != nil && utf8.RuneCountInString(*company.Description) > 3000 {
		return fmt.Errorf("%w: description must be at most 3000 characters", errs.ErrValidation)
	}
	if company.AmountOfEmployees <= 0 {
		return fmt.Errorf("%w: amount of employees must not be negative", errs.ErrValidation)
	}
	if _, err := enum.ParseCompanyType(company.Type); err != nil {
		return fmt.Errorf("%w: invalid type: %v", errs.ErrValidation, err)
	}

	return nil
}
