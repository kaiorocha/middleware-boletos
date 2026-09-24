package domain

import "time"

// Tenant represents an organization using the middleware
type Tenant struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Document    string     `json:"document"`
	Address     string     `json:"address"`
	District    string     `json:"district"`
	City        string     `json:"city"`
	PostalCode  string     `json:"postal_code"`
	State       string     `json:"state"`
	CountryCode string     `json:"country_code"`
	AreaCode    string     `json:"area_code"`
	PhoneNumber string     `json:"phone_number"`
	WebhookURL  string     `json:"webhook_url"`
	OwnerID     *string    `json:"owner_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type TenantAPIToken struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Environment    string     `json:"environment"`
	TokenPrefix    string     `json:"token_prefix"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	Token          string     `json:"token,omitempty"`
	MaskedToken    string     `json:"masked_token,omitempty"`
	EncryptedToken string     `json:"-"`
}

// User represents a system user
type User struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id,omitempty"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	Roles        []string   `json:"roles,omitempty"`
	ExternalID   *string    `json:"external_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	PasswordHash string     `json:"-"`
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
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id,omitempty"`
	Name             string     `json:"name"`
	Type             string     `json:"type,omitempty"`
	Status           string     `json:"status"`
	ExternalID       *string    `json:"external_id,omitempty"`
	Config           *string    `json:"config,omitempty"`
	Metadata         *string    `json:"metadata,omitempty"`
	WebhookToken     string     `json:"webhook_token,omitempty"`
	WebhookTokenHash string     `json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

// TenantProvider enables a provider catalog entry for a tenant.
type TenantProvider struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	ProviderID string     `json:"provider_id"`
	Active     bool       `json:"active"`
	Config     *string    `json:"config,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

// Boleto represents a payment slip (both traditional and proposal)
type Boleto struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	CampaignID     *string    `json:"campaign_id,omitempty"`
	CustomerID     *string    `json:"customer_id,omitempty"`
	RecipientEmail string     `json:"recipient_email"`
	PayerName      string     `json:"payer_name,omitempty"`
	PayerDocument  string     `json:"payer_document,omitempty"`
	ProviderID     *string    `json:"provider_id,omitempty"`
	AmountCents    int64      `json:"amount_cents"`
	DueDate        time.Time  `json:"due_date"`
	Status         string     `json:"status"`
	ExternalID     *string    `json:"external_id,omitempty"`
	Barcode        *string    `json:"barcode,omitempty"`
	DigitableLine  *string    `json:"digitable_line,omitempty"`
	OurNumber      *string    `json:"our_number,omitempty"`
	Base64         *string    `json:"base64,omitempty"`
	IssuedAt       *time.Time `json:"issued_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

const (
	CampaignDraft      = "DRAFT"
	CampaignReady      = "READY"
	CampaignProcessing = "PROCESSING"
	CampaignCompleted  = "COMPLETED"
	CampaignPartial    = "PARTIAL"
	CampaignFailed     = "FAILED"
	CampaignCancelled  = "CANCELLED"
)

type Campaign struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatedBy   *string    `json:"created_by,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CampaignFilters struct {
	TenantID string
	Status   string
	Search   string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

type CampaignImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type CampaignImportPreview struct {
	ImportID         string                `json:"import_id"`
	Checksum         string                `json:"checksum"`
	TotalRows        int                   `json:"total_rows"`
	ValidRows        int                   `json:"valid_rows"`
	InvalidRows      int                   `json:"invalid_rows"`
	TotalAmountCents int64                 `json:"total_amount_cents"`
	Errors           []CampaignImportError `json:"errors"`
	ErrorLimit       int                   `json:"error_limit"`
}

type StagedCampaignRow struct {
	Row        int
	Email      string
	Document   string
	Amount     int64
	DueDate    time.Time
	ExternalID *string
	Field      string
	Code       string
	Message    string
}

type CampaignMetrics struct {
	TotalImported      int         `json:"total_imported"`
	Waiting            int         `json:"waiting"`
	Processing         int         `json:"processing"`
	Issued             int         `json:"issued"`
	Paid               int         `json:"paid"`
	Failed             int         `json:"failed"`
	Blocked            int         `json:"blocked"`
	Cancelled          int         `json:"cancelled"`
	IssuedAmountCents  int64       `json:"issued_amount_cents"`
	PaidAmountCents    int64       `json:"paid_amount_cents"`
	ProcessedPercent   float64     `json:"processed_percent"`
	Conversion         float64     `json:"conversion"`
	AverageTicketCents int64       `json:"average_ticket_cents"`
	ByStatus           []MetricRow `json:"by_status"`
	ByDueDate          []MetricRow `json:"by_due_date"`
	ByAmount           []MetricRow `json:"by_amount"`
}

func (m CampaignMetrics) ByStatusAmount(status string) int64 {
	for _, row := range m.ByStatus {
		if row.ID == status {
			return row.AmountCents
		}
	}
	return 0
}

type CampaignPerformance struct {
	CampaignID         string  `json:"campaign_id"`
	CampaignName       string  `json:"campaign_name"`
	TenantID           string  `json:"tenant_id"`
	Issued             int     `json:"issued"`
	Paid               int     `json:"paid"`
	Conversion         float64 `json:"conversion"`
	RevenueCents       int64   `json:"revenue_cents"`
	AverageTicketCents int64   `json:"average_ticket_cents"`
}

type CampaignDashboard struct {
	Campaigns          int                   `json:"campaigns"`
	Issued             int                   `json:"issued"`
	Paid               int                   `json:"paid"`
	IssuedAmountCents  int64                 `json:"issued_amount_cents"`
	RevenueCents       int64                 `json:"revenue_cents"`
	Conversion         float64               `json:"conversion"`
	AverageTicketCents int64                 `json:"average_ticket_cents"`
	Performance        []CampaignPerformance `json:"performance"`
}

// BoletoFilters scopes dashboard and transaction queries.
type BoletoFilters struct {
	From       *time.Time
	To         *time.Time
	TenantID   string
	ProviderID string
	Status     string
	ExternalID string
	OurNumber  string
	Document   string
	Email      string
	Limit      int
	Offset     int
}

// AdminDashboard aggregates boleto operations across tenants.
type AdminDashboard struct {
	Settlement SettlementDashboard  `json:"settlement"`
	Totals     AdminDashboardTotals `json:"totals"`
	ByTenant   []MetricRow          `json:"by_tenant"`
	ByProvider []MetricRow          `json:"by_provider"`
	ByStatus   []MetricRow          `json:"by_status"`
	Timeline   []TimelineRow        `json:"timeline"`
}

// SettlementDashboard uses the same creation-date cohort as the dashboard totals.
type SettlementDashboard struct {
	Issued           int           `json:"issued"`
	Paid             int           `json:"paid"`
	PaidAmountCents  int64         `json:"paid_amount_cents"`
	UnknownDateCount int           `json:"unknown_date_count"`
	Timeline         []TimelineRow `json:"timeline"`
	ByAmount         []MetricRow   `json:"by_amount"`
}

type AdminDashboardTotals struct {
	Tenants            int     `json:"tenants"`
	Boletos            int     `json:"boletos"`
	AmountCents        int64   `json:"amount_cents"`
	Issued             int     `json:"issued"`
	Paid               int     `json:"paid"`
	Failed             int     `json:"failed"`
	Created            int     `json:"created"`
	Processing         int     `json:"processing"`
	Expired            int     `json:"expired"`
	Cancelled          int     `json:"cancelled"`
	SuccessRate        float64 `json:"success_rate"`
	FailureRate        float64 `json:"failure_rate"`
	AverageTicketCents int64   `json:"average_ticket_cents"`
}

type MetricRow struct {
	ID          string `json:"id,omitempty"`
	Label       string `json:"label"`
	Count       int    `json:"count"`
	AmountCents int64  `json:"amount_cents"`
}

type TimelineRow struct {
	Date        string `json:"date"`
	Count       int    `json:"count"`
	AmountCents int64  `json:"amount_cents"`
}

type BoletoTransaction struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	TenantName       string     `json:"tenant_name"`
	CustomerID       *string    `json:"customer_id,omitempty"`
	CustomerName     *string    `json:"customer_name,omitempty"`
	CustomerDocument *string    `json:"customer_document,omitempty"`
	RecipientEmail   string     `json:"recipient_email"`
	ProviderID       *string    `json:"provider_id,omitempty"`
	ProviderName     *string    `json:"provider_name,omitempty"`
	AmountCents      int64      `json:"amount_cents"`
	DueDate          time.Time  `json:"due_date"`
	Status           string     `json:"status"`
	ExternalID       *string    `json:"external_id,omitempty"`
	OurNumber        *string    `json:"our_number,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	IssuedAt         *time.Time `json:"issued_at,omitempty"`
	DigitableLine    *string    `json:"digitable_line,omitempty"`
	Barcode          *string    `json:"barcode,omitempty"`
	Base64Available  bool       `json:"base64_available"`
	Base64Size       int        `json:"base64_size"`
}

type PaginatedTransactions struct {
	Items  []BoletoTransaction `json:"items"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
	Total  int                 `json:"total"`
}

type TenantProviderConfig struct {
	Provider       Provider       `json:"provider"`
	TenantProvider TenantProvider `json:"tenant_provider"`
}

type OnboardingProviderInput struct {
	ProviderID string  `json:"provider_id"`
	Active     bool    `json:"active"`
	Config     *string `json:"config,omitempty"`
}

type OnboardingInput struct {
	Tenant    Tenant                    `json:"tenant"`
	Admin     *User                     `json:"admin,omitempty"`
	Providers []OnboardingProviderInput `json:"providers,omitempty"`
}

type OnboardingResult struct {
	Tenant    Tenant           `json:"tenant"`
	Admin     *User            `json:"admin,omitempty"`
	Providers []TenantProvider `json:"providers"`
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

// BlacklistEntry blocks boleto emissions for a tenant/document or tenant/email pair.
type BlacklistEntry struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	EntryType       string     `json:"entry_type"`
	Value           string     `json:"value"`
	ValueNormalized string     `json:"value_normalized"`
	Document        string     `json:"document"`
	Name            string     `json:"name"`
	Reason          string     `json:"reason"`
	Notes           *string    `json:"notes,omitempty"`
	Source          string     `json:"source"`
	CreatedBy       *string    `json:"created_by,omitempty"`
	Active          bool       `json:"active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}
