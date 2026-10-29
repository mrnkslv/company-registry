package persistence

import (
	"company-registry/internal/domain/entity"
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Company struct {
	db *sqlx.DB
}

func NewCompany(
	db *sqlx.DB,
) Company {
	return Company{
		db: db,
	}
}

func (c Company) CreateCompany(ctx context.Context, company *entity.Company) error {
	return nil
}

func (c Company) GetCompany(ctx context.Context, id uuid.UUID) (*entity.Company, error) {
	return nil, nil
}

func (c Company) UpdateCompany(ctx context.Context, company *entity.Company) error {
	return nil
}

func (c Company) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	return nil
}
