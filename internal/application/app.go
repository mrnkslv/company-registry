package application

import (
	"context"
	"os"

	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap/zapcore"

	_ "github.com/jackc/pgx/v5/stdlib" // go postgres driver
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"company-registry/internal/application/rest"
	"company-registry/internal/config"
	"company-registry/internal/domain/service"
	"company-registry/internal/infrastructure/persistence"
	"company-registry/migrations"
)

type closeFunc func()

type App struct {
	name           string
	version        string
	cfg            config.Config
	postgresClient *sqlx.DB
	httpServer     *http.Server
	logger         *zap.Logger
}

func New(appVersion string) *App {
	return &App{ //nolint:exhaustruct
		name:    "company-registry",
		version: appVersion,
	}
}

func (app *App) Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	defer stop()

	if err := app.loadConfig(); err != nil {
		return fmt.Errorf("loadConfig: %w", err)
	}

	loggerClose, err := app.createLogger()
	if err != nil {
		return fmt.Errorf("createLogger: %w", err)
	}

	defer loggerClose()

	postgresClose, err := app.connectPostgres(ctx)
	if err != nil {
		return fmt.Errorf("connectPostgres: %w", err)
	}

	defer postgresClose()

	goose.SetBaseFS(migrations.FS)

	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect(app.cfg.Goose.GooseDriver); err != nil {
		return fmt.Errorf("goose.SetDialect: %w", err)
	}

	if err := goose.Up(app.postgresClient.DB, app.cfg.Goose.GooseMigrationDir); err != nil {
		return fmt.Errorf("goose.Up: %w", err)
	}

	if err := app.createHTTPServer(); err != nil {
		return fmt.Errorf("createHTTPServer: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	app.runHTTPServer(ctx, g)

	if err := g.Wait(); err != nil {
		return fmt.Errorf("g.Wait: %w", err)
	}

	return nil
}

func (app *App) loadConfig() error {
	var err error

	if err = godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("godotenv.Load: %w", err)
	}

	app.cfg, err = config.Load()

	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}

	return nil
}

func (app *App) createLogger() (closeFunc, error) {
	var err error

	app.logger, err = newLogger(app.name, app.version, app.cfg.Debug)
	if err != nil {
		return nil, fmt.Errorf("logging.NewLogger: %w", err)
	}

	return func() {
		app.logger.Info("application stopped gracefully")
		app.logger.Sync() //nolint:errcheck
	}, nil
}

func (app *App) connectPostgres(ctx context.Context) (closeFunc, error) {
	var err error

	app.postgresClient, err = sqlx.ConnectContext(ctx, "pgx", app.cfg.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("sqlx.ConnectContext: %w", err)
	}

	app.postgresClient.SetMaxOpenConns(app.cfg.Postgres.MaxOpenConns)
	app.postgresClient.SetMaxIdleConns(app.cfg.Postgres.MaxIdleConns)
	app.postgresClient.SetConnMaxLifetime(app.cfg.Postgres.ConnMaxLifetime)

	app.logger.Info("postgres connected")

	return func() {
		if err := app.postgresClient.Close(); err != nil {
			app.logger.Error("postgresClient.Close", zap.Error(err))
		}
	}, nil
}

func (app *App) createHTTPServer() error { //nolint:funlen
	router := chi.NewRouter()

	httpResponder := rest.NewResponder()

	restServer := rest.NewServer(
		service.NewService(persistence.NewCompany(app.postgresClient)),
		httpResponder,
		app.cfg,
	)

	restServer.RegisterRoutes(
		router,
	)

	app.httpServer = &http.Server{ //nolint:exhaustruct
		Addr:              app.cfg.HTTP.ListenAddress,
		WriteTimeout:      app.cfg.HTTP.WriteTimeout,
		ReadTimeout:       app.cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: app.cfg.HTTP.ReadTimeout,
		IdleTimeout:       app.cfg.HTTP.IdleTimeout,
		Handler:           router,
		MaxHeaderBytes:    1 << 20,
	}

	return nil
}

func (app *App) runHTTPServer(ctx context.Context, g *errgroup.Group) {
	g.Go(func() error {
		go func() {
			<-ctx.Done()

			ctx, cancel := context.WithTimeout(context.Background(), app.cfg.HTTP.ShutdownTimeout)
			defer cancel()

			if err := app.httpServer.Shutdown(ctx); err != nil {
				app.logger.Error("httpServer.Shutdown", zap.Error(err))
			}
		}()

		app.logger.Info("http server started", zap.String("address", app.cfg.HTTP.ListenAddress))

		if err := app.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("httpServer.ListenAndServe: %w", err)
		}

		app.logger.Info("http server stopped")

		return nil
	})
}

func newLogger(appName, appVersion string, debug bool) (*zap.Logger, error) {
	loggerConfig := zap.NewProductionConfig()

	loggerConfig.DisableStacktrace = true

	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	if debug {
		loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("zap.NewProduction: %w", err)
	}

	return logger.With(
		zap.String("FieldAppName", appName),
		zap.String("FieldAppVersion", appVersion),
	), nil
}
