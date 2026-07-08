package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

func TestHealth(t *testing.T) {
	app := &App{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	app.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var payload map[string]map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload["data"]["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", payload)
	}
}

type duplicateUserRepo struct{}

func (r *duplicateUserRepo) Create(*domain.User) error {
	return service.NewDuplicateResource("Já existe um usuário com este e-mail neste tenant.")
}
func (r *duplicateUserRepo) FindByID(string) (*domain.User, error)      { return &domain.User{}, nil }
func (r *duplicateUserRepo) ListByTenant(string) ([]domain.User, error) { return nil, nil }
func (r *duplicateUserRepo) Update(*domain.User) error                  { return nil }
func (r *duplicateUserRepo) Delete(string, string) error                { return nil }

type duplicateCustomerRepo struct{}

func (r *duplicateCustomerRepo) Create(*domain.Customer) error {
	return service.NewDuplicateResource("Já existe um cliente com este documento neste tenant.")
}
func (r *duplicateCustomerRepo) FindByID(string) (*domain.Customer, error) {
	return &domain.Customer{}, nil
}
func (r *duplicateCustomerRepo) ListByTenant(string) ([]domain.Customer, error) {
	return nil, nil
}
func (r *duplicateCustomerRepo) Update(*domain.Customer) error { return nil }
func (r *duplicateCustomerRepo) Delete(string, string) error   { return nil }

type duplicateProviderRepo struct{}

func (r *duplicateProviderRepo) Create(*domain.Provider) error {
	return service.NewDuplicateResource("Já existe um provedor com este nome neste tenant.")
}
func (r *duplicateProviderRepo) FindByID(string) (*domain.Provider, error) {
	return &domain.Provider{}, nil
}
func (r *duplicateProviderRepo) ListByTenant(string) ([]domain.Provider, error) {
	return nil, nil
}
func (r *duplicateProviderRepo) Update(*domain.Provider) error { return nil }
func (r *duplicateProviderRepo) Delete(string, string) error   { return nil }

type duplicateBoletoRepo struct{}

func (r *duplicateBoletoRepo) Create(*domain.Boleto) error {
	return service.NewDuplicateResource("Já existe um boleto com este external_id neste tenant.")
}
func (r *duplicateBoletoRepo) FindByID(string) (*domain.Boleto, error) { return &domain.Boleto{}, nil }
func (r *duplicateBoletoRepo) ListByTenant(string) ([]domain.Boleto, error) {
	return nil, nil
}
func (r *duplicateBoletoRepo) Update(*domain.Boleto) error { return nil }
func (r *duplicateBoletoRepo) Delete(string, string) error { return nil }

func TestCreateHandlersReturnConflictOnDuplicateResource(t *testing.T) {
	validTenantID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerID := "550e8400-e29b-41d4-a716-446655440001"

	tests := []struct {
		name string
		app  *App
		path string
		body string
	}{
		{
			name: "user",
			app:  &App{UserSvc: service.NewUserService(&duplicateUserRepo{})},
			path: "/api/v1/users",
			body: `{"tenant_id":"` + validTenantID + `","email":"user@example.com","name":"User"}`,
		},
		{
			name: "customer",
			app:  &App{CustomerSvc: service.NewCustomerService(&duplicateCustomerRepo{})},
			path: "/api/v1/tenants/" + validTenantID + "/customers",
			body: `{"name":"Customer","document":"123.456.789-00"}`,
		},
		{
			name: "provider",
			app:  &App{ProviderSvc: service.NewProviderService(&duplicateProviderRepo{})},
			path: "/api/v1/tenants/" + validTenantID + "/providers",
			body: `{"name":"Banco Demo"}`,
		},
		{
			name: "boleto",
			app:  &App{BoletoSvc: service.NewBoletoService(&duplicateBoletoRepo{})},
			path: "/api/v1/tenants/" + validTenantID + "/boletos",
			body: `{"customer_id":"` + validCustomerID + `","amount_cents":15000,"due_date":"2026-07-30","status":"CREATED","external_id":"ext-123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			tt.app.routes().ServeHTTP(rr, req)

			if rr.Code != http.StatusConflict {
				t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
			}

			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			if payload.Error.Code != "DUPLICATE_RESOURCE" {
				t.Fatalf("expected DUPLICATE_RESOURCE, got %q", payload.Error.Code)
			}
		})
	}
}
