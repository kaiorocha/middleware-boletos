package moncalieri

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

func TestParseConfig(t *testing.T) {
	cfg, err := parseConfig(`{"base_url":"https://example.com","api_key":"secret","codigo_canal":0,"codigo_cliente":0,"timeout_seconds":5}`)
	if err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
	if cfg.BaseURL != "https://example.com" || cfg.APIKey != "secret" || cfg.TimeoutSeconds != 5 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestIssueBoletoSuccess(t *testing.T) {
	var gotAPIKey string
	var gotPayload envelope[gerarBoletoData]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/CashIn/GerarBoleto" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotAPIKey = r.Header.Get("api-key")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(gerarBoletoResponse{
			Data: gerarBoletoResponseData{
				NossoNumero:    "NN123",
				LinhaDigitavel: "34191.00000 00000.000000 00000.000000 1 12345678901234",
				CodigoBarras:   "3419112345678901234",
				Base64:         "JVBERi0xLjQ=",
			},
		})
	}))
	defer server.Close()

	provider := New(types.ProviderConfig{Name: "Moncalieri", Config: validConfig(server.URL)})
	resp, err := provider.IssueBoleto(context.Background(), validIssueRequest())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gotAPIKey != "test-key" {
		t.Fatalf("expected api-key header, got %q", gotAPIKey)
	}
	if gotPayload.Data.CodigoCanal != 7 || gotPayload.Data.CodigoCliente != 9 {
		t.Fatalf("unexpected channel/client: %+v", gotPayload.Data)
	}
	if gotPayload.Data.DadosSacado.CpfCnpj != 12345678900 || gotPayload.Data.DadosSacado.Cep != 12345678 {
		t.Fatalf("unexpected payer payload: %+v", gotPayload.Data.DadosSacado)
	}
	if !gotPayload.Data.RetornarBase64 || gotPayload.Data.DadosSacado.DdiTerceiro != 55 || gotPayload.Data.DadosSacado.DddTerceiro != 11 || gotPayload.Data.DadosSacado.NumeroCelularTerceiro != 999998888 {
		t.Fatalf("expected base64 and phone fields in payload: %+v", gotPayload.Data)
	}
	if gotPayload.Data.Valor != 123.45 {
		t.Fatalf("expected amount 123.45, got %v", gotPayload.Data.Valor)
	}
	if resp.Status != types.StatusIssued || resp.OurNumber != "NN123" || resp.Barcode == "" || resp.DigitableLine == "" || resp.Base64 != "JVBERi0xLjQ=" || resp.IssuedAt.IsZero() {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestIssueBoletoInvalidConfig(t *testing.T) {
	provider := New(types.ProviderConfig{Name: "Moncalieri", Config: `{}`})
	_, err := provider.IssueBoleto(context.Background(), validIssueRequest())
	assertProviderErrorCode(t, err, errInvalidProviderConfig)
}

func TestIssueBoletoRequiresPayerData(t *testing.T) {
	provider := New(types.ProviderConfig{Name: "Moncalieri", Config: validConfig("https://example.com")})
	req := validIssueRequest()
	req.Payer = nil
	_, err := provider.IssueBoleto(context.Background(), req)
	assertProviderErrorCode(t, err, errInvalidRequest)
}

func TestGetBoletoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/CashIn/ConsultarBoleto" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(consultarBoletoResponse{
			Data: consultarBoletoResponseData{
				Status:               "Liquidado",
				Valor:                12345,
				DataVencimento:       "20260730",
				NossoNumero:          "NN123",
				LinhaDigitavel:       "linha",
				CodigoBarras:         "barra",
				IdentificadorCliente: "boleto-1",
			},
		})
	}))
	defer server.Close()

	provider := New(types.ProviderConfig{Name: "Moncalieri", Config: validConfig(server.URL)})
	got, err := provider.GetBoleto(context.Background(), types.GetRequest{OurNumber: "NN123"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Status != types.StatusPaid || got.OurNumber != "NN123" || got.AmountCents != 12345 {
		t.Fatalf("unexpected boleto summary: %+v", got)
	}
	if got.DueDate.Format("2006-01-02") != "2026-07-30" {
		t.Fatalf("unexpected due date: %s", got.DueDate)
	}
}

func TestCancelBoletoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/CashIn/SolicitarBaixaBoleto" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(solicitarBaixaBoletoResponse{Data: "ok"})
	}))
	defer server.Close()

	provider := New(types.ProviderConfig{Name: "Moncalieri", Config: validConfig(server.URL)})
	got, err := provider.CancelBoleto(context.Background(), types.CancelRequest{OurNumber: "NN123"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Status != types.StatusCancelled || got.OurNumber != "NN123" {
		t.Fatalf("unexpected cancellation response: %+v", got)
	}
}

func TestProviderHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	provider := New(types.ProviderConfig{Name: "Moncalieri", Config: validConfig(server.URL)})
	_, err := provider.GetBoleto(context.Background(), types.GetRequest{OurNumber: "NN123"})
	assertProviderErrorCode(t, err, errProviderHTTP)
}

func TestMapStatus(t *testing.T) {
	tests := map[string]types.BoletoStatus{
		"Pago":       types.StatusPaid,
		"Pagos":      types.StatusPaid,
		"Liquidado":  types.StatusPaid,
		"Pendente":   types.StatusIssued,
		"Registrado": types.StatusIssued,
		"Baixado":    types.StatusCancelled,
		"Cancelado":  types.StatusCancelled,
		"Vencido":    types.StatusExpired,
		"???":        types.StatusProcessing,
	}
	for input, want := range tests {
		if got := MapStatus(input); got != want {
			t.Fatalf("MapStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUnsupportedOperations(t *testing.T) {
	provider := New(types.ProviderConfig{Name: "Moncalieri", Config: validConfig("https://example.com")})
	if err := provider.RegisterWebhook(context.Background(), types.RegisterWebhookRequest{}); err == nil {
		t.Fatal("expected RegisterWebhook error")
	} else {
		assertProviderErrorCode(t, err, errUnsupportedOperation)
	}
	if _, err := provider.ValidateWebhook(context.Background(), types.ValidateWebhookRequest{}); err == nil {
		t.Fatal("expected ValidateWebhook error")
	} else {
		assertProviderErrorCode(t, err, errUnsupportedOperation)
	}
	if _, err := provider.GetBalance(context.Background(), types.BalanceRequest{}); err == nil {
		t.Fatal("expected GetBalance error")
	} else {
		assertProviderErrorCode(t, err, errUnsupportedOperation)
	}
}

func validConfig(baseURL string) string {
	return `{"base_url":"` + baseURL + `","api_key":"test-key","codigo_canal":7,"codigo_cliente":9,"timeout_seconds":5,"instrucoes":"Pagar ate o vencimento."}`
}

func validIssueRequest() types.IssueRequest {
	return types.IssueRequest{
		TenantID:    "tenant-1",
		BoletoID:    "boleto-1",
		CustomerID:  "customer-1",
		AmountCents: 12345,
		DueDate:     time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
		Payer: &types.Payer{
			Document:    "123.456.789-00",
			Name:        "Cliente Demo",
			Address:     "Rua Um, 123",
			District:    "Centro",
			City:        "Sao Paulo",
			PostalCode:  "12345-678",
			State:       "sp",
			Email:       "cliente@example.com",
			CountryCode: "55",
			AreaCode:    "11",
			PhoneNumber: "99999-8888",
		},
	}
}

func assertProviderErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", want)
	}
	perr, ok := err.(*providererrors.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if perr.Code != want {
		t.Fatalf("expected code %s, got %s: %v", want, perr.Code, err)
	}
}
