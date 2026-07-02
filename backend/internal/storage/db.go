package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
	_ "github.com/lib/pq"
)

// Connect opens a DB connection and runs initial migrations
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
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}
	return db, nil
}

func runMigrations(db *sql.DB) error {
	// Simple idempotent table creation for Etapa 2
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			owner_id UUID,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			deleted_at TIMESTAMPTZ
		);`,

		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			email TEXT NOT NULL,
			name TEXT,
			status TEXT NOT NULL DEFAULT 'ACTIVE',
			external_id TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			deleted_at TIMESTAMPTZ
		);`,

		`CREATE TABLE IF NOT EXISTS customers (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			document TEXT,
			status TEXT NOT NULL DEFAULT 'ACTIVE',
			external_id TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			deleted_at TIMESTAMPTZ
		);`,

		`CREATE TABLE IF NOT EXISTS providers (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'ACTIVE',
			external_id TEXT,
			config TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			deleted_at TIMESTAMPTZ
		);`,

		`CREATE TABLE IF NOT EXISTS boletos (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			provider_id UUID REFERENCES providers(id),
			amount_cents BIGINT NOT NULL,
			due_date DATE NOT NULL,
			status TEXT NOT NULL,
			external_id TEXT,
			barcode TEXT,
			digitable_line TEXT,
			our_number TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			deleted_at TIMESTAMPTZ
		);`,

		`CREATE TABLE IF NOT EXISTS webhook_events (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			payload TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,

		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			user_id UUID,
			action TEXT NOT NULL,
			metadata TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ACTIVE';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS external_id TEXT;`,
		`ALTER TABLE customers ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ACTIVE';`,
		`ALTER TABLE customers ADD COLUMN IF NOT EXISTS external_id TEXT;`,
		`ALTER TABLE providers ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ACTIVE';`,
		`ALTER TABLE providers ADD COLUMN IF NOT EXISTS external_id TEXT;`,
		`ALTER TABLE providers ADD COLUMN IF NOT EXISTS config TEXT;`,
		`ALTER TABLE boletos ADD COLUMN IF NOT EXISTS external_id TEXT;`,
		`ALTER TABLE boletos ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'CREATED';`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
