package main

import (
	"testing"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

func TestBoletoValidation(t *testing.T) {
	// setup minimal service with nil repos (we only test validation)
	bs := &service.BoletoService{}
	b := &domain.Boleto{TenantID: "", CustomerID: "", AmountCents: 0, DueDate: time.Time{}}
	if err := bs.Create(b); err == nil { t.Fatalf("expected validation error") }
}
