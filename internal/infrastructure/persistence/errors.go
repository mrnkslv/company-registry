package persistence

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"company-registry/internal/domain/errs"
)

const (
	pgUniqueViolation = "23505"
	pgCheckViolation  = "23514"
)

func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return errs.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return errs.ErrAlreadyExists
		case pgCheckViolation:
			return fmt.Errorf("%w: %s", errs.ErrValidation, pgErr.ConstraintName)
		}
	}

	return err
}
