package webhooks

import (
	"context"

	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/contracts"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/validators"
)

func Receive(ctx context.Context, adapter contracts.ProviderAdapter, req types.ValidateWebhookRequest) (types.WebhookEvent, error) {
	event, err := adapter.ValidateWebhook(ctx, req)
	if err != nil {
		return types.WebhookEvent{}, err
	}
	if err := validators.ValidateWebhookEvent(event); err != nil {
		return types.WebhookEvent{}, err
	}
	return event, nil
}
