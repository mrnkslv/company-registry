//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.uber.org/zap"

	"company-registry/internal/application/rest"
	"company-registry/internal/config"
	"company-registry/internal/domain/service"
	"company-registry/internal/infrastructure/persistence"
	"company-registry/migrations"
)

const (
	testSalt       = "jfghdlkoviojkfdko"
	testSigningKey = "test-signing-key"
	testUsername   = "ituser"
	testPassword   = "secret"
)

var (
	httpServer *httptest.Server
	db         *sqlx.DB
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, database, server, err := setup(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration setup: %v\n", err)
		os.Exit(1)
	}

	db = database
	httpServer = server

	code := m.Run()

	httpServer.Close()
	_ = db.Close()
	_ = container.Terminate(ctx)

	os.Exit(code)
}

func setup(ctx context.Context) (*postgres.PostgresContainer, *sqlx.DB, *httptest.Server, error) {
	container, err := postgres.Run(
		ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("company_registry"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("qwerty"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("postgres.Run: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("container.ConnectionString: %w", err)
	}

	database, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("sqlx.ConnectContext: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		return nil, nil, nil, fmt.Errorf("goose.SetDialect: %w", err)
	}

	if err := goose.Up(database.DB, "."); err != nil {
		return nil, nil, nil, fmt.Errorf("goose.Up: %w", err)
	}

	if _, err := database.ExecContext(
		ctx,
		`INSERT INTO users (name, username, password_hash) VALUES ($1, $2, $3)`,
		"integration tester",
		testUsername,
		passwordHash(testPassword, testSalt),
	); err != nil {
		return nil, nil, nil, fmt.Errorf("insert test user: %w", err)
	}

	return container, database, newTestServer(database), nil
}

func newTestServer(database *sqlx.DB) *httptest.Server {
	responder := rest.NewResponder(zap.NewNop())
	authService := service.NewAuthService(
		persistence.NewUser(database),
		testSalt,
		testSigningKey,
		time.Hour,
	)

	server := rest.NewServer(
		service.NewService(persistence.NewCompany(database)),
		authService,
		rest.NewAuthMiddleware(authService, responder),
		responder,
		config.Config{},
	)

	router := chi.NewRouter()
	server.RegisterRoutes(router)

	return httptest.NewServer(router)
}

func passwordHash(password, salt string) string {
	hash := sha1.New() //nolint:gosec
	hash.Write([]byte(password))

	return fmt.Sprintf("%x", hash.Sum([]byte(salt)))
}

func truncateCompanies(t *testing.T) {
	t.Helper()

	if _, err := db.Exec(`TRUNCATE companies`); err != nil {
		t.Fatalf("truncate companies: %v", err)
	}
}

func login(t *testing.T) string {
	t.Helper()

	status, body := doJSON(t, http.MethodPost, "/auth/v1/login", "", map[string]string{
		"username": testUsername,
		"password": testPassword,
	})
	if status != http.StatusOK {
		t.Fatalf("login: status %d, body %s", status, body)
	}

	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("login: decode token: %v", err)
	}
	if response.Token == "" {
		t.Fatal("login: empty token")
	}

	return response.Token
}

func doJSON(t *testing.T, method, path, token string, payload any) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, httpServer.URL+path, reader)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.Do: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}

	return resp.StatusCode, body
}

func decodeJSON[T any](t *testing.T, body []byte) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("json.Unmarshal: %v, body %s", err, body)
	}

	return value
}

type apiError struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type companyResponse struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Description       *string `json:"description"`
	Type              string  `json:"type"`
	AmountOfEmployees int32   `json:"amount_of_employees"`
	Registered        bool    `json:"registered"`
}
