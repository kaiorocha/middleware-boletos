package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
	_ "github.com/lib/pq"
)

// Connect opens a DB connection and runs versioned migrations
func Connect(cfg *config.Config) (*sql.DB, error) {
	if cfg == nil || strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, fmt.Errorf("database url is required")
	}
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	// pool configuration
	if cfg.DBMaxOpenConns <= 0 {
		cfg.DBMaxOpenConns = 25
	}
	if cfg.DBMaxIdleConns < 0 {
		cfg.DBMaxIdleConns = 5
	}
	if cfg.DBConnMaxLifetimeSeconds <= 0 {
		cfg.DBConnMaxLifetimeSeconds = 1800
	}
	if cfg.DBConnMaxIdleTimeSeconds <= 0 {
		cfg.DBConnMaxIdleTimeSeconds = 300
	}
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeSeconds) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(cfg.DBConnMaxIdleTimeSeconds) * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	if err := runVersionedMigrations(db); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}
	return db, nil
}
