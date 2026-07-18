package events

import (
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

type BoletoCreated struct {
	TenantID string
	BoletoID string
	At       time.Time
}

type BoletoIssued struct {
	TenantID   string
	BoletoID   string
	ProviderID string
	Response   types.IssueResponse
	At         time.Time
}

type BoletoFailed struct {
	TenantID   string
	BoletoID   string
	ProviderID string
	Err        error
	At         time.Time
}

type WebhookReceived struct {
	TenantID   string
	ProviderID string
	Event      types.WebhookEvent
	At         time.Time
}
