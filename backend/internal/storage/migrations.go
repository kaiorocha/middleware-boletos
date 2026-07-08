package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// runVersionedMigrations runs all migrations from the migrations directory
func runVersionedMigrations(db *sql.DB) error {
	// Create schema_migrations table if it doesn't exist
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get the migrations directory
	migrationsDir := filepath.Join(getMigrationsDir())
	if _, err := os.Stat(migrationsDir); err != nil {
		return fmt.Errorf("migrations directory not found: %w", err)
	}

	// List all .sql files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			migrations = append(migrations, file.Name())
		}
	}
	sort.Strings(migrations)

	// Execute each migration
	for _, migFile := range migrations {
		version := strings.TrimSuffix(migFile, ".sql")

		// Check if already executed
		var executed bool
		err := db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			version,
		).Scan(&executed)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if executed {
			continue // Skip already executed migration
		}

		// Read migration file
		content, err := os.ReadFile(filepath.Join(migrationsDir, migFile))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migFile, err)
		}

		// Execute migration
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		// Record migration execution
		if _, err := db.Exec(
			"INSERT INTO schema_migrations (version, executed_at) VALUES ($1, $2)",
			version,
			time.Now(),
		); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}
	}

	return nil
}

// getMigrationsDir returns the path to the migrations directory
func getMigrationsDir() string {
	// Try multiple paths in order of priority
	paths := []string{
		"/app/migrations",                   // Container production path
		"internal/storage/migrations",       // Local development path
		"./internal/storage/migrations",     // Relative path from any working directory
	}

	for _, path := range paths {
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			return path
		}
	}

	// Fallback to the original method if no path works
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "migrations")
}
