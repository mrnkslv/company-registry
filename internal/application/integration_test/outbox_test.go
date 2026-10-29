//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"company-registry/internal/infrastructure/outbox"
)

type outboxEvent struct {
	EventType   string          `db:"event_type"`
	AggregateID string          `db:"aggregate_id"`
	Payload     json.RawMessage `db:"payload"`
	Published   bool            `db:"published"`
}

func truncateOutbox(t *testing.T) {
	t.Helper()

	if _, err := db.Exec(`TRUNCATE outbox_events`); err != nil {
		t.Fatalf("truncate outbox: %v", err)
	}
}

func outboxEvents(t *testing.T) []outboxEvent {
	t.Helper()

	var events []outboxEvent
	if err := db.Select(&events, `
		SELECT event_type, aggregate_id::text AS aggregate_id, payload, published_at IS NOT NULL AS published
		FROM outbox_events
		ORDER BY id`); err != nil {
		t.Fatalf("select outbox: %v", err)
	}

	return events
}

// fakePublisher records published messages and can be told to fail.
type fakePublisher struct {
	mu       sync.Mutex
	messages []outbox.Message
	fail     bool
}

func (p *fakePublisher) Publish(_ context.Context, messages []outbox.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.fail {
		return errors.New("broker unavailable")
	}

	p.messages = append(p.messages, messages...)

	return nil
}

func newRelay(publisher outbox.Publisher, batchSize int) *outbox.Relay {
	return outbox.NewRelay(db, publisher, batchSize, 10*time.Millisecond, zap.NewNop())
}

func TestOutbox_EveryMutationWritesOneEvent(t *testing.T) {
	truncateOutbox(t)

	token := login(t)
	company := createCompany(t, token, validCompany(uniqueName("Evt")))

	status, body := doJSON(t, http.MethodPatch, "/company/v1/"+company.ID, token, map[string]any{"amount_of_employees": 77})
	assertStatus(t, status, http.StatusOK, body)

	status, body = doJSON(t, http.MethodDelete, "/company/v1/"+company.ID, token, nil)
	assertStatus(t, status, http.StatusNoContent, body)

	events := outboxEvents(t)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %+v", len(events), events)
	}

	wantTypes := []string{"company.created", "company.updated", "company.deleted"}
	for i, e := range events {
		if e.EventType != wantTypes[i] || e.AggregateID != company.ID || e.Published {
			t.Fatalf("event %d: %+v", i, e)
		}
	}

	// The update event carries the state after the patch.
	updated := decodeJSON[companyResponse](t, events[1].Payload)
	if updated.AmountOfEmployees != 77 || updated.Name != company.Name {
		t.Fatalf("unexpected update payload %s", events[1].Payload)
	}

	// The delete event carries the last state of the deleted company.
	deleted := decodeJSON[companyResponse](t, events[2].Payload)
	if deleted.ID != company.ID || deleted.AmountOfEmployees != 77 {
		t.Fatalf("unexpected delete payload %s", events[2].Payload)
	}
}

// An event is written in the same transaction as the change, so failed operations leave no events.
func TestOutbox_FailedMutationsWriteNoEvents(t *testing.T) {
	token := login(t)
	existing := createCompany(t, token, validCompany(uniqueName("NoEvt")))
	other := createCompany(t, token, validCompany(uniqueName("NoEvt")))

	truncateOutbox(t)

	failures := []struct {
		name, method, path string
		payload            any
		want               int
	}{
		{"duplicate create", http.MethodPost, "/company/v1/", validCompany(existing.Name), http.StatusConflict},
		{"invalid create", http.MethodPost, "/company/v1/", map[string]any{"name": ""}, http.StatusBadRequest},
		{"patch to taken name", http.MethodPatch, "/company/v1/" + other.ID, map[string]any{"name": existing.Name}, http.StatusConflict},
		{"invalid patch", http.MethodPatch, "/company/v1/" + other.ID, map[string]any{"type": "LLC"}, http.StatusBadRequest},
		{"patch missing", http.MethodPatch, "/company/v1/00000000-0000-0000-0000-000000000001", map[string]any{"registered": true}, http.StatusNotFound},
		{"delete missing", http.MethodDelete, "/company/v1/00000000-0000-0000-0000-000000000001", nil, http.StatusNotFound},
	}

	for _, f := range failures {
		status, body := doJSON(t, f.method, f.path, token, f.payload)
		if status != f.want {
			t.Fatalf("%s: status %d, want %d, body %s", f.name, status, f.want, body)
		}
	}

	if events := outboxEvents(t); len(events) != 0 {
		t.Fatalf("failed operations produced events: %+v", events)
	}
}

func TestRelay_PublishesInOrderAndMarksEvents(t *testing.T) {
	truncateOutbox(t)

	token := login(t)
	company := createCompany(t, token, validCompany(uniqueName("Rel")))

	status, body := doJSON(t, http.MethodPatch, "/company/v1/"+company.ID, token, map[string]any{"registered": false})
	assertStatus(t, status, http.StatusOK, body)

	publisher := &fakePublisher{}

	n, err := newRelay(publisher, 100).PublishBatch(context.Background())
	if err != nil || n != 2 {
		t.Fatalf("PublishBatch: n=%d, err=%v", n, err)
	}

	if len(publisher.messages) != 2 {
		t.Fatalf("published %d messages", len(publisher.messages))
	}

	for i, wantType := range []string{"company.created", "company.updated"} {
		m := publisher.messages[i]
		if string(m.Key) != company.ID || m.Headers["event_type"] != wantType || m.Headers["event_id"] == "" {
			t.Fatalf("message %d: key %s, headers %v", i, m.Key, m.Headers)
		}

		envelope := decodeJSON[struct {
			ID         string          `json:"id"`
			Type       string          `json:"type"`
			OccurredAt time.Time       `json:"occurred_at"`
			Company    companyResponse `json:"company"`
		}](t, m.Value)
		if envelope.Type != wantType || envelope.ID != m.Headers["event_id"] ||
			envelope.OccurredAt.IsZero() || envelope.Company.ID != company.ID {
			t.Fatalf("message %d: unexpected value %s", i, m.Value)
		}
	}

	for _, e := range outboxEvents(t) {
		if !e.Published {
			t.Fatal("event not marked as published")
		}
	}

	// Nothing left: a second run publishes nothing.
	if n, err := newRelay(publisher, 100).PublishBatch(context.Background()); err != nil || n != 0 {
		t.Fatalf("second PublishBatch: n=%d, err=%v", n, err)
	}
}

// If the broker is down, events stay in the outbox and are delivered once it is back.
func TestRelay_KeepsEventsWhenBrokerFails(t *testing.T) {
	truncateOutbox(t)

	token := login(t)
	createCompany(t, token, validCompany(uniqueName("Fail")))

	publisher := &fakePublisher{fail: true}

	if _, err := newRelay(publisher, 100).PublishBatch(context.Background()); err == nil {
		t.Fatal("expected an error from the failing publisher")
	}

	events := outboxEvents(t)
	if len(events) != 1 || events[0].Published {
		t.Fatalf("event must stay unpublished, got %+v", events)
	}

	publisher.fail = false

	if n, err := newRelay(publisher, 100).PublishBatch(context.Background()); err != nil || n != 1 {
		t.Fatalf("retry: n=%d, err=%v", n, err)
	}
}

// Run drains the outbox in batches and stops cleanly when the context is cancelled.
func TestRelay_RunDrainsInBatches(t *testing.T) {
	truncateOutbox(t)

	token := login(t)
	for range 7 {
		createCompany(t, token, validCompany(uniqueName("Batch")))
	}

	publisher := &fakePublisher{}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- newRelay(publisher, 3).Run(ctx) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		publisher.mu.Lock()
		n := len(publisher.messages)
		publisher.mu.Unlock()

		if n == 7 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("relay published %d of 7 events", n)
		}

		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}

// Two relays (e.g. two service replicas) must not publish the same event twice.
func TestRelay_ConcurrentRelaysDoNotDuplicate(t *testing.T) {
	truncateOutbox(t)

	db.SetMaxOpenConns(10)

	token := login(t)
	for range 30 {
		createCompany(t, token, validCompany(uniqueName("Dup")))
	}

	publisher := &fakePublisher{}

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			relay := newRelay(publisher, 5)
			for {
				n, err := relay.PublishBatch(context.Background())
				if err != nil || n == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()

	seen := map[string]bool{}
	for _, m := range publisher.messages {
		id := m.Headers["event_id"]
		if seen[id] {
			t.Fatalf("event %s published twice", id)
		}
		seen[id] = true
	}

	if len(seen) != 30 {
		t.Fatalf("published %d of 30 events", len(seen))
	}
}
