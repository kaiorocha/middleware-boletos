package domain

import "time"

// Tenant represents an organization using the middleware
type Tenant struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	OwnerID   *string    `json:"owner_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// User represents a system user
type User struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Email      string     `json:"email"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	ExternalID *string    `json:"external_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

// Customer represents a customer (cedente/beneficiário)
type Customer struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	Document   *string    `json:"document,omitempty"`
	Email      *string    `json:"email,omitempty"`
	Address    *string    `json:"address,omitempty"`
	Number     *string    `json:"number,omitempty"`
	Complement *string    `json:"complement,omitempty"`
	District   *string    `json:"district,omitempty"`
	City       *string    `json:"city,omitempty"`
	State      *string    `json:"state,omitempty"`
	PostalCode *string    `json:"postal_code,omitempty"`
	Status     string     `json:"status"`
	ExternalID *string    `json:"external_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

// Provider represents a banking provider/integration partner
type Provider struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	ExternalID *string    `json:"external_id,omitempty"`
	Config     *string    `json:"config,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

// Boleto represents a payment slip
type Boleto struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	CustomerID    string     `json:"customer_id"`
	ProviderID    *string    `json:"provider_id,omitempty"`
	AmountCents   int64      `json:"amount_cents"`
	DueDate       time.Time  `json:"due_date"`
	Status        string     `json:"status"`
	ExternalID    *string    `json:"external_id,omitempty"`
	Barcode       *string    `json:"barcode,omitempty"`
	DigitableLine *string    `json:"digitable_line,omitempty"`
	OurNumber     *string    `json:"our_number,omitempty"`
	IssuedAt      *time.Time `json:"issued_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
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
	UserID    *string   `json:"user_id,omitempty"`
	Action    string    `json:"action"`
	Metadata  *string   `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// BlacklistEntry blocks boleto emissions for a tenant/document pair.
type BlacklistEntry struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	Document  string     `json:"document"`
	Name      string     `json:"name"`
	Reason    string     `json:"reason"`
	Notes     *string    `json:"notes,omitempty"`
	Source    string     `json:"source"`
	CreatedBy *string    `json:"created_by,omitempty"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
