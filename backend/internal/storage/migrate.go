package storage

import (
	"database/sql"
)

// RunMigrations executes versioned migrations against the provided DB.
// This function is called explicitly (e.g. via the `migrate` command) and
// is no longer invoked automatically during Connect().
func RunMigrations(db *sql.DB) error {
	return runVersionedMigrations(db)
}
