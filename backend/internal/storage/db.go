package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
	_ "github.com/lib/pq"
)

// Connect opens a DB connection and runs versioned migrations
func Connect(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetConnMaxIdleTime(5 * time.Minute)
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
