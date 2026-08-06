ALTER TABLE users
    ALTER COLUMN tenant_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS password_hash TEXT,
    ADD COLUMN IF NOT EXISTS roles JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_users_lower_email_active
    ON users (lower(email))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_roles_gin
    ON users USING GIN (roles);

