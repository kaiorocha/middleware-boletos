package moncalieri

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

const providerName = "Moncalieri"

type Provider struct {
	cfg    types.ProviderConfig
	config Config
	client *client
}

func New(cfg types.ProviderConfig) *Provider {
	config, err := parseConfig(cfg.Config)
	if err != nil {
		return &Provider{cfg: cfg}
	}
	return &Provider{cfg: cfg, config: config, client: newClient(config)}
}

func (p *Provider) IssueBoleto(ctx context.Context, req types.IssueRequest) (types.IssueResponse, error) {
	if err := p.ensureConfigured(); err != nil {
		return types.IssueResponse{}, err
	}
	payload, err := mapIssueRequest(p.config, req)
	if err != nil {
		return types.IssueResponse{}, err
	}
	var resp gerarBoletoResponse
	if err := p.client.post(ctx, "/api/CashIn/GerarBoleto", payload, &resp); err != nil {
		return types.IssueResponse{}, err
	}
	return mapIssueResponse(req, resp)
}

func (p *Provider) GetBoleto(ctx context.Context, req types.GetRequest) (types.BoletoSummary, error) {
	if err := p.ensureConfigured(); err != nil {
		return types.BoletoSummary{}, err
	}
	if strings.TrimSpace(req.OurNumber) == "" {
		return types.BoletoSummary{}, providererrors.New(errInvalidRequest, "our_number is required", providerName, false)
	}
	payload := envelope[consultarBoletoData]{Data: consultarBoletoData{
		CodigoCanal:   p.config.CodigoCanal,
		CodigoCliente: p.config.CodigoCliente,
		NossoNumero:   strings.TrimSpace(req.OurNumber),
	}}
	var resp consultarBoletoResponse
	if err := p.client.post(ctx, "/api/CashIn/ConsultarBoleto", payload, &resp); err != nil {
		return types.BoletoSummary{}, err
	}
	if err := responseError(resp.ResultCode, resp.Message, resp.ValidationData); err != nil {
		return types.BoletoSummary{}, err
	}
	return mapBoletoSummary(resp.Data), nil
}

func (p *Provider) ListBoletos(ctx context.Context, req types.ListRequest) ([]types.BoletoSummary, error) {
	if err := p.ensureConfigured(); err != nil {
		return nil, err
	}
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -30)
	if req.DateFrom != nil {
		start = *req.DateFrom
	}
	if req.DateTo != nil {
		end = *req.DateTo
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "Todos"
	}
	payload := envelope[consultarBoletoLoteData]{Data: consultarBoletoLoteData{
		CodigoCanal:    p.config.CodigoCanal,
		CodigoCliente:  p.config.CodigoCliente,
		DataInicial:    start.Format(time.RFC3339),
		DataFinal:      end.Format(time.RFC3339),
		TipoPesquisa:   "DataEmissao",
		StatusPesquisa: status,
	}}
	var resp consultarBoletoLoteResponse
	if err := p.client.post(ctx, "/api/CashIn/ConsultarBoletoLote", payload, &resp); err != nil {
		return nil, err
	}
	if err := responseError(resp.ResultCode, resp.Message, resp.ValidationData); err != nil {
		return nil, err
	}
	out := make([]types.BoletoSummary, 0, len(resp.Data))
	for _, item := range resp.Data {
		out = append(out, mapBoletoSummary(item))
	}
	return out, nil
}

func (p *Provider) CancelBoleto(ctx context.Context, req types.CancelRequest) (types.BoletoSummary, error) {
	if err := p.ensureConfigured(); err != nil {
		return types.BoletoSummary{}, err
	}
	if strings.TrimSpace(req.OurNumber) == "" {
		return types.BoletoSummary{}, providererrors.New(errInvalidRequest, "our_number is required", providerName, false)
	}
	payload := envelope[consultarBoletoData]{Data: consultarBoletoData{
		CodigoCanal:   p.config.CodigoCanal,
		CodigoCliente: p.config.CodigoCliente,
		NossoNumero:   strings.TrimSpace(req.OurNumber),
	}}
	var resp solicitarBaixaBoletoResponse
	if err := p.client.post(ctx, "/api/CashIn/SolicitarBaixaBoleto", payload, &resp); err != nil {
		return types.BoletoSummary{}, err
	}
	if err := responseError(resp.ResultCode, resp.Message, resp.ValidationData); err != nil {
		return types.BoletoSummary{}, err
	}
	return types.BoletoSummary{OurNumber: strings.TrimSpace(req.OurNumber), Status: types.StatusCancelled}, nil
}

func (p *Provider) RegisterWebhook(context.Context, types.RegisterWebhookRequest) error {
	return providererrors.New(errUnsupportedOperation, "Moncalieri OpenAPI does not describe webhook registration", providerName, false)
}

func (p *Provider) ValidateWebhook(context.Context, types.ValidateWebhookRequest) (types.WebhookEvent, error) {
	return types.WebhookEvent{}, providererrors.New(errUnsupportedOperation, "Moncalieri OpenAPI does not describe webhook validation", providerName, false)
}

func (p *Provider) GetBalance(context.Context, types.BalanceRequest) (types.BalanceResponse, error) {
	return types.BalanceResponse{}, providererrors.New(errUnsupportedOperation, "Moncalieri OpenAPI does not provide a balance endpoint", providerName, false)
}

func (p *Provider) Health(context.Context) (types.HealthResponse, error) {
	if err := p.ensureConfigured(); err != nil {
		return types.HealthResponse{}, err
	}
	return types.HealthResponse{
		ProviderID: p.cfg.ID,
		Name:       p.cfg.Name,
		Status:     types.HealthOnline,
		Latency:    0,
		Version:    "moncalieri-openapi-v1",
		CheckedAt:  time.Now().UTC(),
	}, nil
}

func (p *Provider) ensureConfigured() error {
	if p.client == nil {
		return providererrors.New(errInvalidProviderConfig, "invalid Moncalieri provider config", providerName, false)
	}
	return validateConfig(p.config)
}

func parseConfig(raw string) (Config, error) {
	var cfg Config
	if strings.TrimSpace(raw) == "" {
		return cfg, providererrors.New(errInvalidProviderConfig, "provider config is required", providerName, false)
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, providererrors.New(errInvalidProviderConfig, "provider config must be valid JSON", providerName, false)
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return providererrors.New(errInvalidProviderConfig, "base_url is required", providerName, false)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return providererrors.New(errInvalidProviderConfig, "api_key is required", providerName, false)
	}
	return nil
}
