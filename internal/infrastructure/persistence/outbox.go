package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"company-registry/internal/domain/entity"
	"company-registry/internal/domain/event"
)

// insertEvent writes an event to the outbox in the same transaction as the change itself,
// so an event exists if and only if the change was committed.
func insertEvent(ctx context.Context, tx *sqlx.Tx, eventType event.Type, company *entity.Company) error {
	payload, err := json.Marshal(event.CompanyFromEntity(company))
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	const query = `
		INSERT INTO outbox_events (event_type, aggregate_id, payload)
		VALUES ($1, $2, $3)`

	if _, err := tx.ExecContext(ctx, query, string(eventType), company.ID, payload); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}

func withTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTxx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}
