# Database Migrations

This directory contains versioned SQL migrations for the middleware-boletos database.

## Structure

Migrations are named with a numeric prefix followed by a descriptive name:
- `001_create_tenants.sql` - Creates the tenants table
- `002_create_users.sql` - Creates the users table
- `003_create_customers.sql` - Creates the customers table
- `004_create_providers.sql` - Creates the providers table
- `005_create_boletos.sql` - Creates the boletos table
- `006_create_webhook_events.sql` - Creates the webhook_events table
- `007_create_audit_logs.sql` - Creates the audit_logs table
- `008_add_status_fields.sql` - Adds status and external_id columns to existing tables

## Migration Execution

Migrations are automatically executed on application startup via `storage/db.go`:

1. When the backend starts, it connects to PostgreSQL
2. Creates a `schema_migrations` table to track executed migrations
3. Reads all `.sql` files from this directory in alphabetical order
4. For each migration:
   - Checks if it's already been executed
   - If not, reads the file and executes the SQL
   - Records the execution timestamp in `schema_migrations`

## Adding New Migrations

1. Create a new file with the next sequential number:
   ```bash
   touch 009_add_new_column.sql
   ```

2. Write your SQL statements:
   ```sql
   ALTER TABLE some_table ADD COLUMN new_column TEXT;
   ```

3. The migration will be automatically executed on next application startup

## Important Notes

- Migration files must be named with a 3-digit numeric prefix followed by underscore and description
- Only `.sql` files are processed
- Migrations are executed in alphabetical order (numerically by prefix)
- Migrations are idempotent when possible (use `IF NOT EXISTS`, `IF NOT` clauses)
- Once a migration is executed, it won't be re-executed (tracked in `schema_migrations` table)
- For container deployments, migrations directory must be present at `/app/migrations`

## Rolling Back

There is no automatic rollback mechanism in this system. To handle schema changes:

1. For reversible changes, create a new migration file that reverts the previous one
2. Always test migrations thoroughly before committing
3. In production, consider managing schema changes manually or with dedicated migration tools for critical operations
