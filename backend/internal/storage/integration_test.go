//go:build integration

package storage

import (
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func integrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("INTEGRATION_DATABASE_URL")
	if dsn == "" {
		t.Skip("INTEGRATION_DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrationsAreCompleteAndIdempotent(t *testing.T) {
	db := integrationDB(t)
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("first migration run: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("idempotent migration run: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count < 26 {
		t.Fatalf("expected at least 26 migrations, got %d", count)
	}
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='providers' AND column_name='webhook_token_hash')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("providers.webhook_token_hash was not created")
	}
}

func TestCustomerUniquenessIsScopedByTenant(t *testing.T) {
	db := integrationDB(t)
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	tenantA, tenantB := uuid.NewString(), uuid.NewString()
	if _, err := db.Exec(`INSERT INTO tenants(id,name) VALUES($1,'A'),($2,'B')`, tenantA, tenantB); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO customers(id,tenant_id,name,document) VALUES($1,$2,'One','12345678900')`, uuid.NewString(), tenantA); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO customers(id,tenant_id,name,document) VALUES($1,$2,'Two','12345678900')`, uuid.NewString(), tenantB); err != nil {
		t.Fatalf("same document in another tenant must be allowed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO customers(id,tenant_id,name,document) VALUES($1,$2,'Duplicate','12345678900')`, uuid.NewString(), tenantA); err == nil {
		t.Fatal("duplicate document in the same tenant must fail")
	}
}
