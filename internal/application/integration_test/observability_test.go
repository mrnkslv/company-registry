//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"company-registry/internal/application/rest"
)

func TestHealth_OK(t *testing.T) {
	rec := httptest.NewRecorder()
	rest.HealthHandler(db).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
}

type failingPinger struct{}

func (failingPinger) PingContext(context.Context) error { return errors.New("connection refused") }

func TestHealth_DatabaseDown(t *testing.T) {
	rec := httptest.NewRecorder()
	rest.HealthHandler(failingPinger{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
}

// A 500 must log the real error but return only a generic message to the client.
func TestInternalError_IsLoggedButNotExposed(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	responder := rest.NewResponder(zap.New(core))

	secret := errors.New(`failed to connect to user=postgres database=company_registry: 10.0.0.5:5432`)

	rec := httptest.NewRecorder()
	responder.ErrorInternal(rec, secret)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}

	body := rec.Body.String()
	if strings.Contains(body, "postgres") || strings.Contains(body, "10.0.0.5") {
		t.Fatalf("internal details leaked to the client: %s", body)
	}

	got := decodeJSON[apiError](t, rec.Body.Bytes())
	if got.Message != "internal server error" {
		t.Fatalf("message %q", got.Message)
	}

	entries := logs.All()
	if len(entries) != 1 || !strings.Contains(entries[0].ContextMap()["error"].(string), "10.0.0.5") {
		t.Fatalf("error was not logged: %+v", entries)
	}
}

// Domain errors are returned without the internal call chain.
func TestErrorMessages_DoNotExposeCallChain(t *testing.T) {
	token := login(t)
	name := uniqueName("Msg")
	createCompany(t, token, validCompany(name))

	status, body := doJSON(t, http.MethodPost, "/company/v1/", token, validCompany(name))
	assertStatus(t, status, http.StatusConflict, body)

	if got := decodeJSON[apiError](t, body); got.Message != "company already exists" {
		t.Fatalf("conflict message %q", got.Message)
	}

	payload := validCompany(uniqueName("Msg"))
	payload["name"] = ""

	status, body = doJSON(t, http.MethodPost, "/company/v1/", token, payload)
	assertStatus(t, status, http.StatusBadRequest, body)

	if got := decodeJSON[apiError](t, body); got.Message != "validation failed: name is required" {
		t.Fatalf("validation message %q", got.Message)
	}

	status, body = doJSON(t, http.MethodPost, "/auth/v1/login", "", map[string]string{"username": "x", "password": "y"})
	assertStatus(t, status, http.StatusUnauthorized, body)

	if got := decodeJSON[apiError](t, body); got.Message != "invalid username or password" {
		t.Fatalf("login message %q", got.Message)
	}
}

// Recoverer turns a panic into a 500 instead of killing the server, and the request is logged.
func TestMiddleware_RecoversPanicsAndLogsRequests(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)

	router := chi.NewRouter()
	router.Use(middleware.RequestID, rest.RequestLogger(zap.New(core)), middleware.Recoverer)
	router.Get("/panic", func(http.ResponseWriter, *http.Request) { panic("boom") })

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}

	entries := logs.FilterMessage("http request").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 request log, got %d", len(entries))
	}

	fields := entries[0].ContextMap()
	if fields["status"] != int64(http.StatusInternalServerError) || fields["path"] != "/panic" || fields["request_id"] == "" {
		t.Fatalf("unexpected log fields %v", fields)
	}
}
