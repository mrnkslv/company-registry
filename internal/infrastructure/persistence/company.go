package persistence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"company-registry/internal/domain/entity"
	"company-registry/internal/domain/event"
)

type Company struct {
	db *sqlx.DB
}

func NewCompany(db *sqlx.DB) Company {
	return Company{db: db}
}

func (c Company) CreateCompany(ctx context.Context, company *entity.Company) error {
	const query = `
		INSERT INTO companies (id, name, description, type, amount_of_employees, registered)
		VALUES (:id, :name, :description, :type, :amount_of_employees, :registered)`

	return withTx(ctx, c.db, func(tx *sqlx.Tx) error {
		if _, err := tx.NamedExecContext(ctx, query, company); err != nil {
			return fmt.Errorf("tx.NamedExecContext: %w", mapError(err))
		}

		return insertEvent(ctx, tx, event.CompanyCreated, company)
	})
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
	const (
		selectQuery = `
			SELECT id, name, description, type, amount_of_employees, registered
			FROM companies
			WHERE id = $1
			FOR UPDATE`

		updateQuery = `
			UPDATE companies
			SET name = :name,
			    description = :description,
			    type = :type,
			    amount_of_employees = :amount_of_employees,
			    registered = :registered
			WHERE id = :id`
	)

	var company entity.Company

	err := withTx(ctx, c.db, func(tx *sqlx.Tx) error {
		if err := tx.GetContext(ctx, &company, selectQuery, id); err != nil {
			return fmt.Errorf("tx.GetContext: %w", mapError(err))
		}

		if err := update(&company); err != nil {
			return err
		}

		if _, err := tx.NamedExecContext(ctx, updateQuery, &company); err != nil {
			return fmt.Errorf("tx.NamedExecContext: %w", mapError(err))
		}

		return insertEvent(ctx, tx, event.CompanyUpdated, &company)
	})
	if err != nil {
		return nil, err
	}

	return &company, nil
}

func (c Company) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	// RETURNING gives the deleted row, so the event carries the last known state.
	const query = `
		DELETE FROM companies
		WHERE id = $1
		RETURNING id, name, description, type, amount_of_employees, registered`

	return withTx(ctx, c.db, func(tx *sqlx.Tx) error {
		var company entity.Company
		if err := tx.GetContext(ctx, &company, query, id); err != nil {
			return fmt.Errorf("tx.GetContext: %w", mapError(err))
		}

		return insertEvent(ctx, tx, event.CompanyDeleted, &company)
	})
}
