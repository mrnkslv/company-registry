package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	HTTP struct {
		ListenAddress   string        `env:"HTTP_LISTEN_ADDRESS,notEmpty"`
		WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"15s"`
		ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s"`
		IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
		ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"15s"`
	}

	Postgres struct {
		DSN             string        `env:"PG_DSN_COMPANY_REGISTRY,notEmpty" json:"-"` // Hide in zap logs
		MaxIdleConns    int           `env:"PG_MAX_IDLE_CONNS" envDefault:"5"`
		MaxOpenConns    int           `env:"PG_MAX_OPEN_CONNS" envDefault:"5"`
		ConnMaxLifetime time.Duration `env:"PG_CONN_MAX_LIFETIME" envDefault:"5m"`
	}

	Goose struct {
		GooseDriver       string `env:"GOOSE_DRIVER" envDefault:"postgres"`
		GooseMigrationDir string `env:"GOOSE_MIGRATION_DIR" envDefault:"./"`
		MigrateUp         bool   `env:"GOOSE_MIGRATION_UP" envDefault:"true"`
	}

	Debug bool `env:"DEBUG" envDefault:"false"`
}

func Load() (Config, error) {
	var config Config

	if err := env.Parse(&config); err != nil {
		return Config{}, fmt.Errorf("env.Parse: %w", err)
	}

	return config, nil
}
