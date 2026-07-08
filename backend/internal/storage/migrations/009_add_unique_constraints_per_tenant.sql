UPDATE users
SET email = lower(trim(email))
WHERE email <> lower(trim(email));

UPDATE customers
SET document = NULLIF(regexp_replace(document, '\D', '', 'g'), '')
WHERE document IS NOT NULL;

UPDATE providers
SET name = trim(name)
WHERE name <> trim(name);

UPDATE boletos
SET external_id = NULLIF(trim(external_id), '')
WHERE external_id IS NOT NULL;

UPDATE boletos
SET our_number = NULLIF(trim(our_number), '')
WHERE our_number IS NOT NULL;

WITH duplicated_users AS (
    SELECT id
    FROM (
        SELECT
            id,
            row_number() OVER (
                PARTITION BY tenant_id, lower(email)
                ORDER BY created_at ASC, id ASC
            ) AS rn
        FROM users
        WHERE deleted_at IS NULL
    ) ranked
    WHERE rn > 1
)
UPDATE users
SET deleted_at = now(), updated_at = now(), status = 'INACTIVE'
WHERE id IN (SELECT id FROM duplicated_users);

WITH duplicated_customers AS (
    SELECT id
    FROM (
        SELECT
            id,
            row_number() OVER (
                PARTITION BY tenant_id, document
                ORDER BY created_at ASC, id ASC
            ) AS rn
        FROM customers
        WHERE document IS NOT NULL AND document <> '' AND deleted_at IS NULL
    ) ranked
    WHERE rn > 1
)
UPDATE customers
SET deleted_at = now(), updated_at = now(), status = 'INACTIVE'
WHERE id IN (SELECT id FROM duplicated_customers);

WITH duplicated_providers AS (
    SELECT id
    FROM (
        SELECT
            id,
            row_number() OVER (
                PARTITION BY tenant_id, lower(name)
                ORDER BY created_at ASC, id ASC
            ) AS rn
        FROM providers
        WHERE deleted_at IS NULL
    ) ranked
    WHERE rn > 1
)
UPDATE providers
SET deleted_at = now(), updated_at = now(), status = 'INACTIVE'
WHERE id IN (SELECT id FROM duplicated_providers);

WITH duplicated_boletos_by_external_id AS (
    SELECT id
    FROM (
        SELECT
            id,
            row_number() OVER (
                PARTITION BY tenant_id, external_id
                ORDER BY created_at ASC, id ASC
            ) AS rn
        FROM boletos
        WHERE external_id IS NOT NULL AND external_id <> '' AND deleted_at IS NULL
    ) ranked
    WHERE rn > 1
)
UPDATE boletos
SET deleted_at = now(), updated_at = now()
WHERE id IN (SELECT id FROM duplicated_boletos_by_external_id);

WITH duplicated_boletos_by_our_number AS (
    SELECT id
    FROM (
        SELECT
            id,
            row_number() OVER (
                PARTITION BY tenant_id, our_number
                ORDER BY created_at ASC, id ASC
            ) AS rn
        FROM boletos
        WHERE our_number IS NOT NULL AND our_number <> '' AND deleted_at IS NULL
    ) ranked
    WHERE rn > 1
)
UPDATE boletos
SET deleted_at = now(), updated_at = now()
WHERE id IN (SELECT id FROM duplicated_boletos_by_our_number);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_lower_email_unique
    ON users (tenant_id, lower(email))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_tenant_document_unique
    ON customers (tenant_id, document)
    WHERE document IS NOT NULL AND document <> '' AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_tenant_lower_name_unique
    ON providers (tenant_id, lower(name))
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_boletos_tenant_external_id_unique
    ON boletos (tenant_id, external_id)
    WHERE external_id IS NOT NULL AND external_id <> '' AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_boletos_tenant_our_number_unique
    ON boletos (tenant_id, our_number)
    WHERE our_number IS NOT NULL AND our_number <> '' AND deleted_at IS NULL;
