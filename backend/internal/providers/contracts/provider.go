package contracts

import (
	"context"

	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

type ProviderAdapter interface {
	IssueBoleto(context.Context, types.IssueRequest) (types.IssueResponse, error)
	GetBoleto(context.Context, types.GetRequest) (types.BoletoSummary, error)
	ListBoletos(context.Context, types.ListRequest) ([]types.BoletoSummary, error)
	CancelBoleto(context.Context, types.CancelRequest) (types.BoletoSummary, error)
	RegisterWebhook(context.Context, types.RegisterWebhookRequest) error
	ValidateWebhook(context.Context, types.ValidateWebhookRequest) (types.WebhookEvent, error)
	GetBalance(context.Context, types.BalanceRequest) (types.BalanceResponse, error)
	Health(context.Context) (types.HealthResponse, error)
}

type ProviderFactory interface {
	Build(types.ProviderConfig) (ProviderAdapter, error)
}
