-- Keeps previously applied campaign staging schemas compatible with the
-- mandatory CPF/CNPJ column introduced in the CSV contract.
ALTER TABLE campaign_import_rows
    ADD COLUMN IF NOT EXISTS payer_document TEXT;
