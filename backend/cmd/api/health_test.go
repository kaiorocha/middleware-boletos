package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/factory"
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

type apiProviderRepo struct {
	item *domain.Provider
}

func (r *apiProviderRepo) Create(*domain.Provider) error { return nil }
func (r *apiProviderRepo) FindByID(string) (*domain.Provider, error) {
	return r.item, nil
}
func (r *apiProviderRepo) ListByTenant(string) ([]domain.Provider, error) { return nil, nil }
func (r *apiProviderRepo) Update(*domain.Provider) error                  { return nil }
func (r *apiProviderRepo) Delete(string, string) error                    { return nil }

type apiCustomerRepo struct {
	item *domain.Customer
}

func (r *apiCustomerRepo) Create(*domain.Customer) error { return nil }
func (r *apiCustomerRepo) FindByID(string) (*domain.Customer, error) {
	return r.item, nil
}
func (r *apiCustomerRepo) ListByTenant(string) ([]domain.Customer, error) { return nil, nil }
func (r *apiCustomerRepo) Update(*domain.Customer) error                  { return nil }
func (r *apiCustomerRepo) Delete(string, string) error                    { return nil }

type apiBoletoRepo struct {
	item    *domain.Boleto
	updated int
}

func (r *apiBoletoRepo) Create(*domain.Boleto) error { return nil }
func (r *apiBoletoRepo) FindByID(string) (*domain.Boleto, error) {
	return r.item, nil
}
func (r *apiBoletoRepo) ListByTenant(string) ([]domain.Boleto, error) { return nil, nil }
func (r *apiBoletoRepo) Update(b *domain.Boleto) error {
	r.updated++
	r.item = b
	return nil
}
func (r *apiBoletoRepo) Delete(string, string) error { return nil }

type apiBlacklistRepo struct {
	item    *domain.BlacklistEntry
	err     error
	blocked bool
	created bool
}

func (r *apiBlacklistRepo) Create(entry *domain.BlacklistEntry) error {
	r.created = true
	r.item = entry
	return r.err
}
func (r *apiBlacklistRepo) FindByID(string, string) (*domain.BlacklistEntry, error) {
	return r.item, r.err
}
func (r *apiBlacklistRepo) FindByDocument(string, string) (*domain.BlacklistEntry, error) {
	return r.item, r.err
}
func (r *apiBlacklistRepo) List(string, string, *bool) ([]domain.BlacklistEntry, error) {
	if r.item == nil {
		return nil, r.err
	}
	return []domain.BlacklistEntry{*r.item}, r.err
}
func (r *apiBlacklistRepo) Update(entry *domain.BlacklistEntry) error {
	r.item = entry
	return r.err
}
func (r *apiBlacklistRepo) SoftDelete(string, string) error { return r.err }
func (r *apiBlacklistRepo) IsBlocked(string, string) (*domain.BlacklistEntry, bool, error) {
	return r.item, r.blocked, r.err
}

func completeAPICustomer(tenantID string) *domain.Customer {
	return &domain.Customer{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		TenantID:   tenantID,
		Name:       "Cliente Demo",
		Document:   apiStringPtr("123.456.789-00"),
		Email:      apiStringPtr("cliente@example.com"),
		Address:    apiStringPtr("Rua Um"),
		Number:     apiStringPtr("123"),
		District:   apiStringPtr("Centro"),
		City:       apiStringPtr("Sao Paulo"),
		State:      apiStringPtr("SP"),
		PostalCode: apiStringPtr("12345-678"),
		Status:     "ACTIVE",
	}
}

func TestBlacklistCheckRouteReturnsRawBlockedPayload(t *testing.T) {
	validTenantID := "550e8400-e29b-41d4-a716-446655440000"
	app := &App{
		BlacklistSvc: service.NewBlacklistService(&apiBlacklistRepo{
			blocked: true,
			item: &domain.BlacklistEntry{
				ID:       "550e8400-e29b-41d4-a716-446655440010",
				TenantID: validTenantID,
				Document: "12345678900",
				Reason:   "Solicitação do cliente",
				Active:   true,
			},
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+validTenantID+"/blacklist/check?document=123.456.789-00", nil)
	authorizeTenant(req, validTenantID)
	rr := httptest.NewRecorder()
	app.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Blocked bool   `json:"blocked"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !payload.Blocked || payload.Reason != "Solicitação do cliente" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestBlacklistCreateDuplicateReturnsConflict(t *testing.T) {
	validTenantID := "550e8400-e29b-41d4-a716-446655440000"
	app := &App{
		BlacklistSvc: service.NewBlacklistService(&apiBlacklistRepo{
			err: service.NewDuplicateResource("Este documento já está bloqueado neste tenant."),
		}),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+validTenantID+"/blacklist", strings.NewReader(`{"document":"12345678900","reason":"Solicitação do cliente"}`))
	req.Header.Set("Content-Type", "application/json")
	authorizeTenant(req, validTenantID)
	rr := httptest.NewRecorder()
	app.routes().ServeHTTP(rr, req)

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
}

func apiStringPtr(value string) *string {
	return &value
}

func authorizeTenant(req *http.Request, tenantID string) {
	req.Header.Set("X-User-ID", "550e8400-e29b-41d4-a716-446655449999")
	req.Header.Set("X-Tenant-ID", tenantID)
}

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
			if strings.Contains(tt.path, "/api/v1/tenants/") {
				authorizeTenant(req, validTenantID)
			}
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

func TestEmitBoletoRouteUsesTenantBoletoHandler(t *testing.T) {
	validTenantID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderID := "550e8400-e29b-41d4-a716-446655440002"
	validBoletoID := "550e8400-e29b-41d4-a716-446655440003"

	providerRepo := &apiProviderRepo{item: &domain.Provider{
		ID:       validProviderID,
		TenantID: validTenantID,
		Name:     "Mock",
		Status:   "ACTIVE",
	}}
	boletoRepo := &apiBoletoRepo{item: &domain.Boleto{
		ID:          validBoletoID,
		TenantID:    validTenantID,
		CustomerID:  validCustomerID,
		ProviderID:  &validProviderID,
		AmountCents: 15000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
	}}
	providerFactory := factory.NewProviderFactory()
	app := &App{
		ProviderSvc: service.NewProviderService(providerRepo),
		BoletoSvc: service.NewBoletoService(boletoRepo).
			WithCustomerRepository(&apiCustomerRepo{item: completeAPICustomer(validTenantID)}).
			WithProviderRepository(providerRepo).
			WithBlacklistService(service.NewBlacklistService(&apiBlacklistRepo{})).
			WithProviderFactory(providerFactory),
		Factory: providerFactory,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+validTenantID+"/boletos/"+validBoletoID+"/emit", nil)
	authorizeTenant(req, validTenantID)
	rr := httptest.NewRecorder()

	app.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data domain.Boleto `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if payload.Data.Status != "ISSUED" {
		t.Fatalf("expected ISSUED, got %q", payload.Data.Status)
	}
	if payload.Data.Barcode == nil || payload.Data.DigitableLine == nil || payload.Data.OurNumber == nil || payload.Data.IssuedAt == nil {
		t.Fatalf("expected emitted boleto fields, got %+v", payload.Data)
	}
}

func TestEmitBoletoRouteReturnsCustomerBlocked(t *testing.T) {
	validTenantID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderID := "550e8400-e29b-41d4-a716-446655440002"
	validBoletoID := "550e8400-e29b-41d4-a716-446655440003"

	boletoRepo := &apiBoletoRepo{item: &domain.Boleto{
		ID:          validBoletoID,
		TenantID:    validTenantID,
		CustomerID:  validCustomerID,
		ProviderID:  &validProviderID,
		AmountCents: 15000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
	}}
	blacklistSvc := service.NewBlacklistService(&apiBlacklistRepo{
		blocked: true,
		item: &domain.BlacklistEntry{
			ID:       "550e8400-e29b-41d4-a716-446655440010",
			TenantID: validTenantID,
			Document: "12345678900",
			Reason:   "Solicitação do cliente",
			Active:   true,
		},
	})
	providerFactory := factory.NewProviderFactory()
	app := &App{
		BoletoSvc: service.NewBoletoService(boletoRepo).
			WithCustomerRepository(&apiCustomerRepo{item: completeAPICustomer(validTenantID)}).
			WithProviderRepository(&apiProviderRepo{}).
			WithBlacklistService(blacklistSvc).
			WithProviderFactory(providerFactory),
		Factory: providerFactory,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+validTenantID+"/boletos/"+validBoletoID+"/emit", nil)
	authorizeTenant(req, validTenantID)
	rr := httptest.NewRecorder()
	app.routes().ServeHTTP(rr, req)

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
	if payload.Error.Code != "CUSTOMER_BLOCKED" {
		t.Fatalf("expected CUSTOMER_BLOCKED, got %q", payload.Error.Code)
	}
	if boletoRepo.updated != 0 {
		t.Fatalf("expected no boleto update/provider flow, got %d updates", boletoRepo.updated)
	}
}

func TestTenantScopedAuthorization(t *testing.T) {
	tenantA := "550e8400-e29b-41d4-a716-446655440000"
	tenantB := "550e8400-e29b-41d4-a716-446655440099"
	app := &App{
		CustomerSvc:  service.NewCustomerService(&apiCustomerRepo{}),
		BoletoSvc:    service.NewBoletoService(&apiBoletoRepo{}),
		BlacklistSvc: service.NewBlacklistService(&apiBlacklistRepo{}),
	}

	tests := []struct {
		name          string
		path          string
		allowedTenant string
		userID        string
		want          int
	}{
		{
			name:          "customer allowed",
			path:          "/api/v1/tenants/" + tenantA + "/customers",
			allowedTenant: tenantA,
			userID:        "550e8400-e29b-41d4-a716-446655449999",
			want:          http.StatusOK,
		},
		{
			name:          "customer forbidden",
			path:          "/api/v1/tenants/" + tenantB + "/customers",
			allowedTenant: tenantA,
			userID:        "550e8400-e29b-41d4-a716-446655449999",
			want:          http.StatusForbidden,
		},
		{
			name: "customer unauthenticated",
			path: "/api/v1/tenants/" + tenantA + "/customers",
			want: http.StatusUnauthorized,
		},
		{
			name:          "customer invalid user",
			path:          "/api/v1/tenants/" + tenantA + "/customers",
			allowedTenant: tenantA,
			userID:        "not-a-uuid",
			want:          http.StatusUnauthorized,
		},
		{
			name:          "boleto allowed",
			path:          "/api/v1/tenants/" + tenantA + "/boletos",
			allowedTenant: tenantA,
			userID:        "550e8400-e29b-41d4-a716-446655449999",
			want:          http.StatusOK,
		},
		{
			name:          "boleto forbidden",
			path:          "/api/v1/tenants/" + tenantB + "/boletos",
			allowedTenant: tenantA,
			userID:        "550e8400-e29b-41d4-a716-446655449999",
			want:          http.StatusForbidden,
		},
		{
			name:          "blacklist allowed",
			path:          "/api/v1/tenants/" + tenantA + "/blacklist",
			allowedTenant: tenantA,
			userID:        "550e8400-e29b-41d4-a716-446655449999",
			want:          http.StatusOK,
		},
		{
			name:          "blacklist forbidden",
			path:          "/api/v1/tenants/" + tenantB + "/blacklist",
			allowedTenant: tenantA,
			userID:        "550e8400-e29b-41d4-a716-446655449999",
			want:          http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			if tt.allowedTenant != "" {
				req.Header.Set("X-Tenant-ID", tt.allowedTenant)
			}
			rr := httptest.NewRecorder()
			app.routes().ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("expected %d, got %d: %s", tt.want, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestProviderHealthBalanceAndWebhookRoutes(t *testing.T) {
	validTenantID := "550e8400-e29b-41d4-a716-446655440000"
	validProviderID := "550e8400-e29b-41d4-a716-446655440002"
	providerRepo := &apiProviderRepo{item: &domain.Provider{
		ID:       validProviderID,
		TenantID: validTenantID,
		Name:     "Mock",
		Status:   "ACTIVE",
	}}
	providerFactory := factory.NewProviderFactory()
	app := &App{
		ProviderSvc: service.NewProviderService(providerRepo),
		Factory:     providerFactory,
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "health",
			method: http.MethodGet,
			path:   "/api/v1/providers/health?tenant_id=" + validTenantID + "&provider_id=" + validProviderID,
		},
		{
			name:   "balance",
			method: http.MethodGet,
			path:   "/api/v1/providers/balance?tenant_id=" + validTenantID + "&provider_id=" + validProviderID,
		},
		{
			name:   "webhook",
			method: http.MethodPost,
			path:   "/api/v1/providers/webhook?tenant_id=" + validTenantID + "&provider_id=" + validProviderID,
			body:   `{"type":"boleto.paid","status":"PAID"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			authorizeTenant(req, validTenantID)
			rr := httptest.NewRecorder()

			app.routes().ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}
