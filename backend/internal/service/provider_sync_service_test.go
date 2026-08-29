package service

import (
	"testing"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	providertypes "github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

func TestApplyProviderSummaryCompletesRegisteredBoletoWithoutRequiringBase64(t *testing.T) {
	boleto := &domain.Boleto{Status: string(providertypes.StatusProcessing)}
	updated := applyProviderSummary(boleto, providertypes.BoletoSummary{
		Status:        providertypes.StatusIssued,
		OurNumber:     "123",
		Barcode:       "barcode",
		DigitableLine: "line",
	})
	if !updated || boleto.Status != string(providertypes.StatusIssued) || boleto.DigitableLine == nil || boleto.Base64 != nil || boleto.IssuedAt == nil {
		t.Fatalf("unexpected synchronized boleto: %+v", boleto)
	}
}

func TestApplyProviderSummaryDoesNotReportUnchangedBoleto(t *testing.T) {
	ourNumber, barcode, line := "123", "barcode", "line"
	boleto := &domain.Boleto{Status: string(providertypes.StatusIssued), OurNumber: &ourNumber, Barcode: &barcode, DigitableLine: &line}
	now := boleto.CreatedAt
	boleto.IssuedAt = &now
	if applyProviderSummary(boleto, providertypes.BoletoSummary{Status: providertypes.StatusIssued, OurNumber: ourNumber, Barcode: barcode, DigitableLine: line}) {
		t.Fatal("unchanged provider response must not emit an update")
	}
}
