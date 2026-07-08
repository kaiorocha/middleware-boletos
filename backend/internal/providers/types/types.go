package types

import "time"

type BoletoStatus string

const (
	StatusCreated    BoletoStatus = "CREATED"
	StatusProcessing BoletoStatus = "PROCESSING"
	StatusIssued     BoletoStatus = "ISSUED"
	StatusFailed     BoletoStatus = "FAILED"
	StatusCancelled  BoletoStatus = "CANCELLED"
	StatusPartial    BoletoStatus = "PARTIAL"
	StatusPaid       BoletoStatus = "PAID"
	StatusExpired    BoletoStatus = "EXPIRED"
)

type ProviderConfig struct {
	ID       string
	TenantID string
	Name     string
	Config   string
}

type IssueRequest struct {
	TenantID     string
	BoletoID     string
	CustomerID   string
	ExternalID   string
	AmountCents  int64
	DueDate      time.Time
	Payer        *Payer
	Instructions string
}

type Payer struct {
	Document   string
	Name       string
	Address    string
	District   string
	City       string
	PostalCode string
	State      string
	Email      string
}

type IssueResponse struct {
	ExternalID    string       `json:"external_id"`
	Barcode       string       `json:"barcode"`
	DigitableLine string       `json:"digitable_line"`
	OurNumber     string       `json:"our_number"`
	Status        BoletoStatus `json:"status"`
	IssuedAt      time.Time    `json:"issued_at"`
}

type GetRequest struct {
	TenantID   string
	ProviderID string
	ExternalID string
	OurNumber  string
}

type ListRequest struct {
	TenantID   string
	ProviderID string
	DateFrom   *time.Time
	DateTo     *time.Time
	Status     string
}

type CancelRequest struct {
	TenantID   string
	ProviderID string
	ExternalID string
	OurNumber  string
}

type RegisterWebhookRequest struct {
	TenantID   string
	ProviderID string
	URL        string
	Secret     string
}

type ValidateWebhookRequest struct {
	ProviderID string
	Headers    map[string]string
	Body       []byte
}

type BalanceRequest struct {
	TenantID   string
	ProviderID string
}

type BoletoSummary struct {
	ExternalID    string       `json:"external_id"`
	OurNumber     string       `json:"our_number"`
	Status        BoletoStatus `json:"status"`
	AmountCents   int64        `json:"amount_cents"`
	DueDate       time.Time    `json:"due_date"`
	Barcode       string       `json:"barcode,omitempty"`
	DigitableLine string       `json:"digitable_line,omitempty"`
}

type WebhookEvent struct {
	ID         string       `json:"id"`
	ProviderID string       `json:"provider_id"`
	TenantID   string       `json:"tenant_id"`
	Type       string       `json:"type"`
	BoletoID   string       `json:"boleto_id,omitempty"`
	ExternalID string       `json:"external_id,omitempty"`
	OurNumber  string       `json:"our_number,omitempty"`
	Status     BoletoStatus `json:"status,omitempty"`
	Payload    []byte       `json:"payload,omitempty"`
	ReceivedAt time.Time    `json:"received_at"`
}

type BalanceResponse struct {
	ProviderID     string    `json:"provider_id"`
	Currency       string    `json:"currency"`
	AvailableCents int64     `json:"available_cents"`
	BlockedCents   int64     `json:"blocked_cents"`
	CheckedAt      time.Time `json:"checked_at"`
}

type HealthStatus string

const (
	HealthOnline  HealthStatus = "ONLINE"
	HealthOffline HealthStatus = "OFFLINE"
)

type HealthResponse struct {
	ProviderID string        `json:"provider_id"`
	Name       string        `json:"name"`
	Status     HealthStatus  `json:"status"`
	Latency    time.Duration `json:"latency"`
	Version    string        `json:"version"`
	CheckedAt  time.Time     `json:"checked_at"`
}
