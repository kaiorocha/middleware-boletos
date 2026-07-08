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
