package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sola-scriptura-search-api/pkg/schema/config"
)

var (
	pgDB   *sqlx.DB
	pgOnce sync.Once
	pgMu   sync.RWMutex
)

// postgresEnabled tracks whether Postgres was initialized
var postgresEnabled bool

// InitPostgres initializes the PostgreSQL database connection.
func InitPostgres(ctx context.Context) error {
	var initErr error
	pgOnce.Do(func() {
		cfg := config.GetConfig()

		if cfg.PostgresURI == "" {
			initErr = fmt.Errorf("POSTGRES_URI is required")
			return
		}

		var err error
		pgDB, err = sqlx.ConnectContext(ctx, "postgres", cfg.PostgresURI)
		if err != nil {
			initErr = fmt.Errorf("failed to connect to PostgreSQL: %w", err)
			return
		}

		// Configure connection pool
		pgDB.SetMaxOpenConns(25)
		pgDB.SetMaxIdleConns(25)
		pgDB.SetConnMaxLifetime(5 * time.Minute)
		pgDB.SetConnMaxIdleTime(1 * time.Minute)

		// Verify connectivity
		if err := pgDB.PingContext(ctx); err != nil {
			initErr = fmt.Errorf("failed to ping PostgreSQL: %w", err)
			return
		}

		postgresEnabled = true
	})
	return initErr
}

// PostgresEnabled returns whether Postgres is available
func PostgresEnabled() bool {
	return postgresEnabled
}

// GetPostgres returns the PostgreSQL database instance
func GetPostgres() *sqlx.DB {
	pgMu.RLock()
	defer pgMu.RUnlock()
	return pgDB
}

// ClosePostgres closes the PostgreSQL database connection
func ClosePostgres() error {
	pgMu.Lock()
	defer pgMu.Unlock()
	if pgDB != nil {
		return pgDB.Close()
	}
	return nil
}
