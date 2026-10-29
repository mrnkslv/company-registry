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

	Auth struct {
		Salt       string        `env:"AUTH_SALT,notEmpty" json:"-"`
		SigningKey string        `env:"AUTH_SIGNING_KEY,notEmpty" json:"-"`
		TokenTTL   time.Duration `env:"AUTH_TOKEN_TTL" envDefault:"12h"`
	}

	Goose struct {
		GooseDriver       string `env:"GOOSE_DRIVER" envDefault:"postgres"`
		GooseMigrationDir string `env:"GOOSE_MIGRATION_DIR" envDefault:"."`
		MigrateUp         bool   `env:"GOOSE_MIGRATION_UP" envDefault:"true"`
	}

	Kafka struct {
		Brokers []string `env:"KAFKA_BROKERS,notEmpty" envSeparator:","`
		Topic   string   `env:"KAFKA_TOPIC" envDefault:"company-events"`
	}

	Outbox struct {
		PollInterval time.Duration `env:"OUTBOX_POLL_INTERVAL" envDefault:"1s"`
		BatchSize    int           `env:"OUTBOX_BATCH_SIZE" envDefault:"100"`
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
