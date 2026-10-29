package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Message is what the relay hands to the broker.
type Message struct {
	Key     []byte
	Value   []byte
	Headers map[string]string
}

type Publisher interface {
	Publish(ctx context.Context, messages []Message) error
}

// envelope is the JSON published to Kafka.
type envelope struct {
	ID         uuid.UUID       `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Company    json.RawMessage `json:"company"`
}

type row struct {
	ID          int64           `db:"id"`
	EventID     uuid.UUID       `db:"event_id"`
	EventType   string          `db:"event_type"`
	AggregateID uuid.UUID       `db:"aggregate_id"`
	Payload     json.RawMessage `db:"payload"`
	CreatedAt   time.Time       `db:"created_at"`
}

// Relay periodically moves unpublished events from the outbox table to the broker.
// Delivery is at-least-once: if publishing succeeds but marking fails, the batch is sent
// again, so consumers should deduplicate by the event id.
type Relay struct {
	db        *sqlx.DB
	publisher Publisher
	batchSize int
	interval  time.Duration
	logger    *zap.Logger
}

func NewRelay(db *sqlx.DB, publisher Publisher, batchSize int, interval time.Duration, logger *zap.Logger) *Relay {
	return &Relay{
		db:        db,
		publisher: publisher,
		batchSize: batchSize,
		interval:  interval,
		logger:    logger,
	}
}

// Run publishes events until ctx is canceled. Errors are logged and retried on the next tick.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		if err := r.drain(ctx); err != nil && ctx.Err() == nil {
			r.logger.Error("outbox relay", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Relay) drain(ctx context.Context) error {
	for {
		n, err := r.PublishBatch(ctx)
		if err != nil {
			return err
		}

		if n < r.batchSize {
			return nil
		}
	}
}

// PublishBatch publishes one batch of events and returns how many were published.
// FOR UPDATE SKIP LOCKED lets several service instances run relays without sending
// the same event twice.
func (r *Relay) PublishBatch(ctx context.Context) (int, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("db.BeginTxx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	const selectQuery = `
		SELECT id, event_id, event_type, aggregate_id, payload, created_at
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`

	var rows []row
	if err := tx.SelectContext(ctx, &rows, selectQuery, r.batchSize); err != nil {
		return 0, fmt.Errorf("select outbox events: %w", err)
	}

	if len(rows) == 0 {
		return 0, nil
	}

	messages := make([]Message, 0, len(rows))
	ids := make([]int64, 0, len(rows))

	for _, e := range rows {
		value, err := json.Marshal(envelope{
			ID:         e.EventID,
			Type:       e.EventType,
			OccurredAt: e.CreatedAt,
			Company:    e.Payload,
		})
		if err != nil {
			return 0, fmt.Errorf("json.Marshal: %w", err)
		}

		messages = append(messages, Message{
			// Same key -> same partition, so events of one company stay in order.
			Key:   []byte(e.AggregateID.String()),
			Value: value,
			Headers: map[string]string{
				"event_id":   e.EventID.String(),
				"event_type": e.EventType,
			},
		})
		ids = append(ids, e.ID)
	}

	if err := r.publisher.Publish(ctx, messages); err != nil {
		return 0, fmt.Errorf("publisher.Publish: %w", err)
	}

	const markQuery = `UPDATE outbox_events SET published_at = now() WHERE id = ANY($1)`
	if _, err := tx.ExecContext(ctx, markQuery, ids); err != nil {
		return 0, fmt.Errorf("mark outbox events: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("tx.Commit: %w", err)
	}

	return len(rows), nil
}
