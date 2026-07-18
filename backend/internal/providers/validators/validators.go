package validators

import (
	"errors"

	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

func ValidateIssueRequest(req types.IssueRequest) error {
	if req.TenantID == "" || req.BoletoID == "" || req.CustomerID == "" {
		return errors.New("missing boleto identifiers")
	}
	if req.AmountCents <= 0 {
		return errors.New("amount_cents must be greater than zero")
	}
	if req.DueDate.IsZero() {
		return errors.New("due_date is required")
	}
	return nil
}

func ValidateWebhookEvent(event types.WebhookEvent) error {
	if event.TenantID == "" || event.ProviderID == "" || event.Type == "" {
		return errors.New("missing webhook event fields")
	}
	return nil
}
