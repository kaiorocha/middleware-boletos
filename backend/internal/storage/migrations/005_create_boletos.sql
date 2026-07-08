CREATE TABLE IF NOT EXISTS boletos (
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
);
