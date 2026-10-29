//nolint:errcheck, gosec
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"company-registry/internal/domain/entity"
	"company-registry/internal/domain/errs"
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
	const query = `
		INSERT INTO companies (id, name, description, type, amount_of_employees, registered)
		VALUES (:id, :name, :description, :type, :amount_of_employees, :registered)`

	if _, err := c.db.NamedExecContext(ctx, query, company); err != nil {
		return fmt.Errorf("db.NamedExecContext: %w", mapError(err))
	}

	return nil
}

func (c Company) GetCompany(ctx context.Context, id uuid.UUID) (*entity.Company, error) {
	const query = `
		SELECT id, name, description, type, amount_of_employees, registered
		FROM companies
		WHERE id = $1`

	var company entity.Company
	if err := c.db.GetContext(ctx, &company, query, id); err != nil {
		return nil, fmt.Errorf("db.GetContext: %w", mapError(err))
	}

	return &company, nil
}

func (c Company) UpdateCompany(
	ctx context.Context,
	id uuid.UUID,
	update func(*entity.Company) error,
) (*entity.Company, error) {
	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("db.BeginTxx: %w", err)
	}
	defer func() { tx.Rollback() }()

	var company entity.Company

	const selectQuery = `
		SELECT id, name, description, type, amount_of_employees, registered
		FROM companies
		WHERE id = $1
		FOR UPDATE`

	if err := tx.GetContext(ctx, &company, selectQuery, id); err != nil {
		return nil, fmt.Errorf("tx.GetContext: %w", mapError(err))
	}

	if err := update(&company); err != nil {
		return nil, err
	}

	const updateQuery = `
		UPDATE companies
		SET name = :name,
		    description = :description,
		    type = :type,
		    amount_of_employees = :amount_of_employees,
		    registered = :registered
		WHERE id = :id`

	if _, err := tx.NamedExecContext(ctx, updateQuery, &company); err != nil {
		return nil, fmt.Errorf("tx.NamedExecContext: %w", mapError(err))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("tx.Commit: %w", err)
	}

	return &company, nil
}

func (c Company) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM companies WHERE id = $1`

	res, err := c.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("db.ExecContext: %w", mapError(err))
	}

	return checkAffected(res)
}

func checkAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("RowsAffected: %w", err)
	}

	if n == 0 {
		return errs.ErrNotFound
	}

	return nil
}
