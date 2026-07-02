package domain

import "time"

// Tenant represents an organization using the middleware
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OwnerID   string    `json:"owner_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User represents a system user
type User struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Customer represents a customer (cedente/beneficiário)
type Customer struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Document  string    `json:"document,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Provider represents a banking provider/integration partner
type Provider struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Config    string    `json:"config,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Boleto represents a payment slip
type Boleto struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	CustomerID   string    `json:"customer_id"`
	ProviderID   string    `json:"provider_id,omitempty"`
	AmountCents  int64     `json:"amount_cents"`
	DueDate      time.Time `json:"due_date"`
	Barcode      string    `json:"barcode,omitempty"`
	OurNumber    string    `json:"our_number,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WebhookEvent represents events received from providers or emitted to clients
type WebhookEvent struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditLog stores auditable actions
type AuditLog struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id,omitempty"`
	Action    string    `json:"action"`
	Metadata  string    `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
