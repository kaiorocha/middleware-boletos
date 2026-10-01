ALTER TABLE tenant_api_tokens
    ADD COLUMN IF NOT EXISTS encrypted_token TEXT;
