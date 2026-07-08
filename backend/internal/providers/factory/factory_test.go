package factory

import (
	"context"
	"testing"

	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

func TestFactoryBuildsMockProvider(t *testing.T) {
	adapter, err := NewProviderFactory().Build(types.ProviderConfig{Name: "Mock"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	health, err := adapter.Health(context.Background())
	if err != nil {
		t.Fatalf("expected health, got %v", err)
	}
	if health.Status != types.HealthOnline {
		t.Fatalf("expected online health, got %q", health.Status)
	}
}

func TestFactoryRejectsUnknownProvider(t *testing.T) {
	_, err := NewProviderFactory().Build(types.ProviderConfig{Name: "Unknown"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
