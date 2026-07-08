package mock

import (
	"context"
	"testing"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

func TestMockProviderIssueBoleto(t *testing.T) {
	adapter := New(types.ProviderConfig{ID: "provider-1", TenantID: "tenant-1", Name: "Mock"})
	resp, err := adapter.IssueBoleto(context.Background(), types.IssueRequest{
		TenantID:    "tenant-1",
		BoletoID:    "boleto-1",
		CustomerID:  "customer-1",
		AmountCents: 15000,
		DueDate:     time.Now().AddDate(0, 0, 7),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Status != types.StatusIssued {
		t.Fatalf("expected ISSUED, got %q", resp.Status)
	}
	if resp.ExternalID == "" || resp.Barcode == "" || resp.DigitableLine == "" || resp.OurNumber == "" {
		t.Fatalf("expected fake boleto fields, got %+v", resp)
	}
}

func TestMockProviderHealth(t *testing.T) {
	adapter := New(types.ProviderConfig{ID: "provider-1", Name: "Mock"})
	health, err := adapter.Health(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if health.Status != types.HealthOnline || health.Version == "" {
		t.Fatalf("expected online health with version, got %+v", health)
	}
}

func TestMockProviderValidateWebhook(t *testing.T) {
	adapter := New(types.ProviderConfig{ID: "provider-1", TenantID: "tenant-1", Name: "Mock"})
	event, err := adapter.ValidateWebhook(context.Background(), types.ValidateWebhookRequest{
		Body: []byte(`{"type":"boleto.paid","status":"PAID"}`),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if event.ID == "" || event.ProviderID != "provider-1" || event.TenantID != "tenant-1" {
		t.Fatalf("expected normalized webhook event, got %+v", event)
	}
}
