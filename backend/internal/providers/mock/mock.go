package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

type Provider struct {
	cfg   types.ProviderConfig
	delay time.Duration
}

type config struct {
	DelayMS int `json:"delay_ms"`
}

func New(cfg types.ProviderConfig) *Provider {
	p := &Provider{cfg: cfg}
	if cfg.Config != "" {
		var c config
		if err := json.Unmarshal([]byte(cfg.Config), &c); err == nil && c.DelayMS > 0 {
			p.delay = time.Duration(c.DelayMS) * time.Millisecond
		}
	}
	return p
}

func (p *Provider) IssueBoleto(ctx context.Context, req types.IssueRequest) (types.IssueResponse, error) {
	if p.delay > 0 {
		timer := time.NewTimer(p.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return types.IssueResponse{}, ctx.Err()
		case <-timer.C:
		}
	}

	id := uuid.New().String()
	digits := onlyDigits(id)
	return types.IssueResponse{
		ExternalID:    "mock-" + id,
		Barcode:       "34191" + padRight(digits, 39),
		DigitableLine: fmt.Sprintf("34191.%s %s.%s %s.%s 1 %013d", digits[0:5], digits[5:10], digits[10:15], digits[15:20], digits[20:25], req.AmountCents),
		OurNumber:     "MOCK" + digits[0:10],
		Status:        types.StatusIssued,
		IssuedAt:      time.Now().UTC(),
	}, nil
}

func (p *Provider) GetBoleto(context.Context, types.GetRequest) (types.BoletoSummary, error) {
	return types.BoletoSummary{Status: types.StatusIssued}, nil
}

func (p *Provider) ListBoletos(context.Context, types.ListRequest) ([]types.BoletoSummary, error) {
	return []types.BoletoSummary{}, nil
}

func (p *Provider) CancelBoleto(context.Context, types.CancelRequest) (types.BoletoSummary, error) {
	return types.BoletoSummary{Status: types.StatusCancelled}, nil
}

func (p *Provider) RegisterWebhook(context.Context, types.RegisterWebhookRequest) error {
	return nil
}

func (p *Provider) ValidateWebhook(_ context.Context, req types.ValidateWebhookRequest) (types.WebhookEvent, error) {
	if len(req.Body) == 0 {
		return types.WebhookEvent{}, providererrors.New("INVALID_WEBHOOK", "empty webhook payload", p.cfg.Name, false)
	}
	var event types.WebhookEvent
	if err := json.Unmarshal(req.Body, &event); err != nil {
		return types.WebhookEvent{}, providererrors.New("INVALID_WEBHOOK", "invalid webhook payload", p.cfg.Name, false)
	}
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.ProviderID == "" {
		event.ProviderID = p.cfg.ID
	}
	if event.TenantID == "" {
		event.TenantID = p.cfg.TenantID
	}
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now().UTC()
	}
	event.Payload = req.Body
	return event, nil
}

func (p *Provider) GetBalance(context.Context, types.BalanceRequest) (types.BalanceResponse, error) {
	return types.BalanceResponse{
		ProviderID:     p.cfg.ID,
		Currency:       "BRL",
		AvailableCents: 100000000,
		BlockedCents:   0,
		CheckedAt:      time.Now().UTC(),
	}, nil
}

func (p *Provider) Health(ctx context.Context) (types.HealthResponse, error) {
	start := time.Now()
	if p.delay > 0 {
		timer := time.NewTimer(p.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return types.HealthResponse{}, ctx.Err()
		case <-timer.C:
		}
	}
	return types.HealthResponse{
		ProviderID: p.cfg.ID,
		Name:       p.cfg.Name,
		Status:     types.HealthOnline,
		Latency:    time.Since(start),
		Version:    "mock-v1",
		CheckedAt:  time.Now().UTC(),
	}, nil
}

func onlyDigits(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			out = append(out, s[i])
		}
	}
	return padRight(string(out), 25)
}

func padRight(s string, size int) string {
	for len(s) < size {
		s += "0"
	}
	if len(s) > size {
		return s[:size]
	}
	return s
}
