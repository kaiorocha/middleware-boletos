package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/factory"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

const (
	testJWTSecret = "test-secret-with-enough-entropy"
	testJWTIssuer = "middleware-boletos-tests"
	testJWTAud    = "middleware-boletos-api"
	testUserID    = "550e8400-e29b-41d4-a716-446655449999"
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
func (r *duplicateUserRepo) FindByEmail(string) (*domain.User, error)   { return &domain.User{}, nil }
func (r *duplicateUserRepo) HasRole(string) (bool, error)               { return false, nil }
func (r *duplicateUserRepo) ListByTenant(string) ([]domain.User, error) { return nil, nil }
func (r *duplicateUserRepo) Update(*domain.User) error                  { return nil }
func (r *duplicateUserRepo) Delete(string, string) error                { return nil }

type apiUserRepo struct {
	item    *domain.User
	created []*domain.User
	hasRole bool
	err     error
}

func (r *apiUserRepo) Create(user *domain.User) error {
	r.created = append(r.created, user)
	r.item = user
	return r.err
}
func (r *apiUserRepo) FindByID(string) (*domain.User, error) {
	if r.item != nil {
		return r.item, r.err
	}
	return nil, r.err
}
func (r *apiUserRepo) FindByEmail(email string) (*domain.User, error) {
	if r.item != nil && strings.EqualFold(r.item.Email, email) {
		return r.item, r.err
	}
	return nil, service.ErrValidation
}
func (r *apiUserRepo) HasRole(string) (bool, error) { return r.hasRole, r.err }
func (r *apiUserRepo) ListByTenant(string) ([]domain.User, error) {
	if r.item == nil {
		return nil, r.err
	}
	return []domain.User{*r.item}, r.err
}
func (r *apiUserRepo) Update(*domain.User) error   { return r.err }
func (r *apiUserRepo) Delete(string, string) error { return r.err }

type apiTenantRepo struct {
	items   map[string]*domain.Tenant
	created *domain.Tenant
	deleted string
}

func (r *apiTenantRepo) Create(t *domain.Tenant) error {
	if t.ID == "" {
		t.ID = "550e8400-e29b-41d4-a716-446655440077"
	}
	r.created = t
	if r.items == nil {
		r.items = map[string]*domain.Tenant{}
	}
	r.items[t.ID] = t
	return nil
}
func (r *apiTenantRepo) FindByID(id string) (*domain.Tenant, error) {
	if item, ok := r.items[id]; ok {
		return item, nil
	}
	return nil, service.ErrValidation
}
func (r *apiTenantRepo) List() ([]domain.Tenant, error) {
	out := make([]domain.Tenant, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, *item)
	}
	return out, nil
}
func (r *apiTenantRepo) Update(*domain.Tenant) error { return nil }
func (r *apiTenantRepo) Delete(id string) error      { r.deleted = id; delete(r.items, id); return nil }

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
func (r *duplicateProviderRepo) ListCatalog() ([]domain.Provider, error) {
	return nil, nil
}
func (r *duplicateProviderRepo) Update(*domain.Provider) error { return nil }
func (r *duplicateProviderRepo) Delete(string, string) error   { return nil }
func (r *duplicateProviderRepo) AssignToTenant(tenantID, providerID string, active bool, config *string) (*domain.TenantProvider, error) {
	return &domain.TenantProvider{TenantID: tenantID, ProviderID: providerID, Active: active, Config: config}, nil
}
func (r *duplicateProviderRepo) IsAllowedForTenant(string, string) (bool, error) { return true, nil }

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
func (r *apiProviderRepo) ListCatalog() ([]domain.Provider, error)        { return nil, nil }
func (r *apiProviderRepo) Update(*domain.Provider) error                  { return nil }
func (r *apiProviderRepo) Delete(string, string) error                    { return nil }
func (r *apiProviderRepo) AssignToTenant(tenantID, providerID string, active bool, config *string) (*domain.TenantProvider, error) {
	return &domain.TenantProvider{TenantID: tenantID, ProviderID: providerID, Active: active, Config: config}, nil
}
func (r *apiProviderRepo) IsAllowedForTenant(string, string) (bool, error) { return true, nil }

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
func (r *apiBoletoRepo) AdminDashboard(domain.BoletoFilters) (*domain.AdminDashboard, error) {
	return &domain.AdminDashboard{Totals: domain.AdminDashboardTotals{Tenants: 1, Boletos: 1, Issued: 1, AmountCents: 15000}}, nil
}
func (r *apiBoletoRepo) ListTransactions(filters domain.BoletoFilters) (*domain.PaginatedTransactions, error) {
	return &domain.PaginatedTransactions{
		Items:  []domain.BoletoTransaction{{ID: "550e8400-e29b-41d4-a716-446655440003", TenantID: "550e8400-e29b-41d4-a716-446655440000", TenantName: "Tenant Demo", AmountCents: 15000, Status: "ISSUED", CreatedAt: time.Now()}},
		Limit:  filters.Limit,
		Offset: filters.Offset,
		Total:  1,
	}, nil
}

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
	app := authenticatedTestApp(&App{
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
	})

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
	app := authenticatedTestApp(&App{
		BlacklistSvc: service.NewBlacklistService(&apiBlacklistRepo{
			err: service.NewDuplicateResource("Este documento já está bloqueado neste tenant."),
		}),
	})

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
	authorizeTenants(req, tenantID)
}

func authorizeTenants(req *http.Request, tenantIDs ...string) {
	req.Header.Set("Authorization", "Bearer "+testJWT(testUserID, tenantIDs, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret))
}

func authorizePlatformAdmin(req *http.Request, tenantIDs ...string) {
	req.Header.Set("Authorization", "Bearer "+testJWTWithRoles(testUserID, tenantIDs, []string{authn.RolePlatformAdmin}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret))
}

func authenticatedTestApp(app *App) *App {
	validator, err := authn.NewHMACValidator(authn.JWTConfig{
		Secret:   testJWTSecret,
		Issuer:   testJWTIssuer,
		Audience: testJWTAud,
	})
	if err != nil {
		panic(err)
	}
	app.Authenticator = NewRequestAuthenticator("production", validator)
	app.Authorizer = NewIdentityTenantAuthorizer()
	app.TokenIssuer = validator
	return app
}

func developmentTestApp(app *App) *App {
	app.Authenticator = NewRequestAuthenticator("development", nil)
	app.Authorizer = NewIdentityTenantAuthorizer()
	return app
}

func testJWT(userID string, tenantIDs []string, exp time.Time, issuer, audience, secret string) string {
	return testJWTWithRoles(userID, tenantIDs, nil, exp, issuer, audience, secret)
}

func testJWTWithRoles(userID string, tenantIDs []string, roles []string, exp time.Time, issuer, audience, secret string) string {
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	claims := map[string]any{
		"sub":        userID,
		"tenant_ids": tenantIDs,
		"exp":        exp.Unix(),
	}
	if roles != nil {
		claims["roles"] = roles
	}
	if len(tenantIDs) == 1 {
		claims["tenant_id"] = tenantIDs[0]
	}
	if issuer != "" {
		claims["iss"] = issuer
	}
	if audience != "" {
		claims["aud"] = audience
	}
	unsigned := encodeJWTPart(header) + "." + encodeJWTPart(claims)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeJWTPart(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func TestGlobalTenantRoutesRequirePlatformAdmin(t *testing.T) {
	tenantID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &apiTenantRepo{items: map[string]*domain.Tenant{
		tenantID: {ID: tenantID, Name: "Tenant A"},
	}}
	app := authenticatedTestApp(&App{TenantSvc: service.NewTenantService(repo)})

	tests := []struct {
		name   string
		method string
		body   string
		token  string
		want   int
	}{
		{name: "GET without token", method: http.MethodGet, want: http.StatusUnauthorized},
		{
			name:   "GET authenticated without platform admin",
			method: http.MethodGet,
			token:  testJWT(testUserID, []string{tenantID}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret),
			want:   http.StatusForbidden,
		},
		{
			name:   "GET invalid role",
			method: http.MethodGet,
			token:  testJWTWithRoles(testUserID, []string{tenantID}, []string{"TENANT_ADMIN"}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret),
			want:   http.StatusForbidden,
		},
		{
			name:   "GET lowercase platform admin role normalized",
			method: http.MethodGet,
			token:  testJWTWithRoles(testUserID, []string{tenantID}, []string{"platform_admin"}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret),
			want:   http.StatusOK,
		},
		{
			name:   "GET platform admin",
			method: http.MethodGet,
			token:  testJWTWithRoles(testUserID, []string{tenantID}, []string{authn.RolePlatformAdmin}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret),
			want:   http.StatusOK,
		},
		{name: "POST without token", method: http.MethodPost, body: `{"name":"Tenant Novo"}`, want: http.StatusUnauthorized},
		{
			name:   "POST authenticated without platform admin",
			method: http.MethodPost,
			body:   `{"name":"Tenant Novo"}`,
			token:  testJWT(testUserID, []string{tenantID}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret),
			want:   http.StatusForbidden,
		},
		{
			name:   "POST invalid role",
			method: http.MethodPost,
			body:   `{"name":"Tenant Novo"}`,
			token:  testJWTWithRoles(testUserID, []string{tenantID}, []string{"ADMIN"}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret),
			want:   http.StatusForbidden,
		},
		{
			name:   "POST platform admin",
			method: http.MethodPost,
			body:   `{"name":"Tenant Novo"}`,
			token:  testJWTWithRoles(testUserID, []string{tenantID}, []string{authn.RolePlatformAdmin}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret),
			want:   http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/tenants", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rr := httptest.NewRecorder()
			app.routes().ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("expected %d, got %d: %s", tt.want, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestMyTenantsReturnsOnlyClaimTenants(t *testing.T) {
	tenantA := "550e8400-e29b-41d4-a716-446655440000"
	tenantB := "550e8400-e29b-41d4-a716-446655440099"
	tenantC := "550e8400-e29b-41d4-a716-446655440088"
	app := authenticatedTestApp(&App{TenantSvc: service.NewTenantService(&apiTenantRepo{items: map[string]*domain.Tenant{
		tenantA: {ID: tenantA, Name: "Tenant A"},
		tenantB: {ID: tenantB, Name: "Tenant B"},
		tenantC: {ID: tenantC, Name: "Tenant C"},
	}})})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/tenants", nil)
	authorizeTenants(req, tenantA, tenantB)
	rr := httptest.NewRecorder()
	app.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Data []domain.Tenant `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(payload.Data) != 2 {
		t.Fatalf("expected 2 tenants, got %+v", payload.Data)
	}
	for _, tenant := range payload.Data {
		if tenant.ID == tenantC {
			t.Fatalf("tenant outside claims was returned: %+v", payload.Data)
		}
	}
}

func TestLogin(t *testing.T) {
	tenantID := "550e8400-e29b-41d4-a716-446655440000"
	hash, err := authn.HashPassword("Senha123456!")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user := &domain.User{
		ID:           testUserID,
		TenantID:     tenantID,
		Email:        "admin@example.com",
		Name:         "Admin",
		Status:       "ACTIVE",
		Roles:        []string{authn.RoleTenantAdmin},
		PasswordHash: hash,
	}
	app := authenticatedTestApp(&App{UserSvc: service.NewUserService(&apiUserRepo{item: user})})

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "valid login", body: `{"email":"admin@example.com","password":"Senha123456!"}`, want: http.StatusOK},
		{name: "wrong password", body: `{"email":"admin@example.com","password":"errada"}`, want: http.StatusUnauthorized},
		{name: "unknown user", body: `{"email":"missing@example.com","password":"Senha123456!"}`, want: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			app.routes().ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("expected %d, got %d: %s", tt.want, rr.Code, rr.Body.String())
			}
			if tt.want == http.StatusOK {
				var payload struct {
					Data struct {
						AccessToken string `json:"access_token"`
						User        struct {
							Roles     []string `json:"roles"`
							TenantIDs []string `json:"tenant_ids"`
						} `json:"user"`
					} `json:"data"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
					t.Fatalf("invalid json: %v", err)
				}
				if payload.Data.AccessToken == "" || len(payload.Data.User.TenantIDs) != 1 {
					t.Fatalf("expected token and tenant ids, got %+v", payload.Data)
				}
			}
		})
	}
}

func TestLoginCORSPreflight(t *testing.T) {
	app := authenticatedTestApp(&App{})
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	rr := httptest.NewRecorder()

	app.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 preflight, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS headers")
	}
}

func TestLoginTokenAccessesProtectedTenantRoute(t *testing.T) {
	tenantID := "550e8400-e29b-41d4-a716-446655440000"
	hash, err := authn.HashPassword("Senha123456!")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user := &domain.User{
		ID:           testUserID,
		TenantID:     tenantID,
		Email:        "tenant@example.com",
		Name:         "Tenant Admin",
		Status:       "ACTIVE",
		Roles:        []string{authn.RoleTenantAdmin},
		PasswordHash: hash,
	}
	app := authenticatedTestApp(&App{
		UserSvc:     service.NewUserService(&apiUserRepo{item: user}),
		CustomerSvc: service.NewCustomerService(&apiCustomerRepo{}),
	})

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"tenant@example.com","password":"Senha123456!"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	app.routes().ServeHTTP(loginRR, loginReq)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", loginRR.Code, loginRR.Body.String())
	}

	var loginPayload struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRR.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("invalid login json: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+tenantID+"/customers", nil)
	req.Header.Set("Authorization", "Bearer "+loginPayload.Data.AccessToken)
	rr := httptest.NewRecorder()
	app.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected protected route 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBootstrapPlatformAdminDevelopment(t *testing.T) {
	repo := &apiUserRepo{}
	cfg := &config.Config{
		Env:                    "development",
		BootstrapAdminEmail:    "admin@middleware.local",
		BootstrapAdminPassword: "ChangeMe123456!",
		BootstrapAdminName:     "Administrador",
	}
	err := bootstrapPlatformAdmin(cfg, service.NewUserService(repo))
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if len(repo.created) != 1 || repo.created[0].PasswordHash == "" || !(authn.Identity{Roles: repo.created[0].Roles}).HasRole(authn.RolePlatformAdmin) {
		t.Fatalf("expected platform admin created securely, got %+v", repo.created)
	}

	repo.hasRole = true
	err = bootstrapPlatformAdmin(cfg, service.NewUserService(repo))
	if err != nil {
		t.Fatalf("bootstrap idempotent: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected idempotent bootstrap, created=%d", len(repo.created))
	}
}

func TestBootstrapPlatformAdminProductionPolicy(t *testing.T) {
	validCfg := func() *config.Config {
		return &config.Config{
			Env:                    "production",
			EnableAdminBootstrap:   true,
			BootstrapAdminEmail:    "admin@middleware.local",
			BootstrapAdminPassword: "ChangeMe123456!",
			BootstrapAdminName:     "Administrador",
		}
	}

	tests := []struct {
		name       string
		cfg        *config.Config
		wantCreate bool
		wantErr    bool
	}{
		{
			name: "production without ENABLE_ADMIN_BOOTSTRAP",
			cfg: &config.Config{
				Env:                    "production",
				BootstrapAdminEmail:    "admin@middleware.local",
				BootstrapAdminPassword: "ChangeMe123456!",
				BootstrapAdminName:     "Administrador",
			},
		},
		{
			name: "production with ENABLE_ADMIN_BOOTSTRAP false",
			cfg: &config.Config{
				Env:                    "production",
				EnableAdminBootstrap:   false,
				BootstrapAdminEmail:    "admin@middleware.local",
				BootstrapAdminPassword: "ChangeMe123456!",
				BootstrapAdminName:     "Administrador",
			},
		},
		{
			name:       "production with ENABLE_ADMIN_BOOTSTRAP true and complete config",
			cfg:        validCfg(),
			wantCreate: true,
		},
		{
			name: "production with ENABLE_ADMIN_BOOTSTRAP true and incomplete config",
			cfg: &config.Config{
				Env:                  "production",
				EnableAdminBootstrap: true,
				BootstrapAdminEmail:  "admin@middleware.local",
			},
			wantErr: true,
		},
		{
			name: "production with ENABLE_ADMIN_BOOTSTRAP true and weak password",
			cfg: &config.Config{
				Env:                    "production",
				EnableAdminBootstrap:   true,
				BootstrapAdminEmail:    "admin@middleware.local",
				BootstrapAdminPassword: "short",
				BootstrapAdminName:     "Administrador",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &apiUserRepo{}
			err := bootstrapPlatformAdmin(tt.cfg, service.NewUserService(repo))
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got %v", tt.wantErr, err)
			}
			if (len(repo.created) == 1) != tt.wantCreate {
				t.Fatalf("expected created=%v, got %d", tt.wantCreate, len(repo.created))
			}
		})
	}
}

func TestPlatformAdminCreatesTenantAdmin(t *testing.T) {
	tenantRepo := &apiTenantRepo{}
	userRepo := &apiUserRepo{}
	app := authenticatedTestApp(&App{
		TenantSvc: service.NewTenantService(tenantRepo),
		UserSvc:   service.NewUserService(userRepo),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tenants", strings.NewReader(`{
		"name":"Cliente Demonstração",
		"admin":{"name":"Cliente Admin","email":"cliente@demo.local","password":"Cliente123456!"}
	}`))
	req.Header.Set("Content-Type", "application/json")
	authorizePlatformAdmin(req)
	rr := httptest.NewRecorder()
	app.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if tenantRepo.created == nil || len(userRepo.created) != 1 {
		t.Fatalf("expected tenant and admin created")
	}
	admin := userRepo.created[0]
	if admin.TenantID != tenantRepo.created.ID || admin.PasswordHash == "" || !(authn.Identity{Roles: admin.Roles}).HasRole(authn.RoleTenantAdmin) {
		t.Fatalf("unexpected tenant admin: %+v", admin)
	}
}

func TestCreateHandlersReturnConflictOnDuplicateResource(t *testing.T) {
	validTenantID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerID := "550e8400-e29b-41d4-a716-446655440001"

	tests := []struct {
		name     string
		app      *App
		path     string
		body     string
		platform bool
	}{
		{
			name: "user",
			app:  authenticatedTestApp(&App{UserSvc: service.NewUserService(&duplicateUserRepo{})}),
			path: "/api/v1/users",
			body: `{"tenant_id":"` + validTenantID + `","email":"user@example.com","name":"User"}`,
		},
		{
			name: "customer",
			app:  authenticatedTestApp(&App{CustomerSvc: service.NewCustomerService(&duplicateCustomerRepo{})}),
			path: "/api/v1/tenants/" + validTenantID + "/customers",
			body: `{"name":"Customer","document":"123.456.789-00"}`,
		},
		{
			name:     "provider",
			app:      authenticatedTestApp(&App{ProviderSvc: service.NewProviderService(&duplicateProviderRepo{})}),
			path:     "/api/v1/admin/providers",
			body:     `{"name":"Banco Demo"}`,
			platform: true,
		},
		{
			name: "boleto",
			app:  authenticatedTestApp(&App{BoletoSvc: service.NewBoletoService(&duplicateBoletoRepo{})}),
			path: "/api/v1/tenants/" + validTenantID + "/boletos",
			body: `{"customer_id":"` + validCustomerID + `","amount_cents":15000,"due_date":"2026-07-30","status":"CREATED","external_id":"ext-123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.platform {
				authorizePlatformAdmin(req)
			} else {
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

func TestAdminDashboardAndTransactionsRequirePlatformAdmin(t *testing.T) {
	app := authenticatedTestApp(&App{BoletoSvc: service.NewBoletoService(&apiBoletoRepo{})})
	tests := []struct {
		name       string
		path       string
		authorize  func(*http.Request)
		wantStatus int
	}{
		{name: "dashboard unauthenticated", path: "/api/v1/admin/dashboard", wantStatus: http.StatusUnauthorized},
		{name: "dashboard tenant user forbidden", path: "/api/v1/admin/dashboard", authorize: func(req *http.Request) { authorizeTenant(req, "550e8400-e29b-41d4-a716-446655440000") }, wantStatus: http.StatusForbidden},
		{name: "dashboard platform admin", path: "/api/v1/admin/dashboard", authorize: func(req *http.Request) { authorizePlatformAdmin(req) }, wantStatus: http.StatusOK},
		{name: "transactions platform admin", path: "/api/v1/admin/transactions?limit=10", authorize: func(req *http.Request) { authorizePlatformAdmin(req) }, wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authorize != nil {
				tt.authorize(req)
			}
			rr := httptest.NewRecorder()

			app.routes().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, rr.Code, rr.Body.String())
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
	app := authenticatedTestApp(&App{
		ProviderSvc: service.NewProviderService(providerRepo),
		BoletoSvc: service.NewBoletoService(boletoRepo).
			WithCustomerRepository(&apiCustomerRepo{item: completeAPICustomer(validTenantID)}).
			WithProviderRepository(providerRepo).
			WithBlacklistService(service.NewBlacklistService(&apiBlacklistRepo{})).
			WithProviderFactory(providerFactory),
		Factory: providerFactory,
	})

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
	app := authenticatedTestApp(&App{
		BoletoSvc: service.NewBoletoService(boletoRepo).
			WithCustomerRepository(&apiCustomerRepo{item: completeAPICustomer(validTenantID)}).
			WithProviderRepository(&apiProviderRepo{}).
			WithBlacklistService(blacklistSvc).
			WithProviderFactory(providerFactory),
		Factory: providerFactory,
	})

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
	app := authenticatedTestApp(&App{
		CustomerSvc:  service.NewCustomerService(&apiCustomerRepo{}),
		BoletoSvc:    service.NewBoletoService(&apiBoletoRepo{}),
		BlacklistSvc: service.NewBlacklistService(&apiBlacklistRepo{}),
	})

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
			if tt.userID != "" && tt.allowedTenant != "" {
				req.Header.Set("Authorization", "Bearer "+testJWT(tt.userID, []string{tt.allowedTenant}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret))
			}
			rr := httptest.NewRecorder()
			app.routes().ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("expected %d, got %d: %s", tt.want, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestJWTAuthentication(t *testing.T) {
	tenantA := "550e8400-e29b-41d4-a716-446655440000"
	path := "/api/v1/tenants/" + tenantA + "/customers"
	app := authenticatedTestApp(&App{CustomerSvc: service.NewCustomerService(&apiCustomerRepo{})})

	tests := []struct {
		name      string
		token     string
		want      int
		noAuth    bool
		malformed bool
	}{
		{
			name:  "valid token",
			token: testJWT(testUserID, []string{tenantA}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret),
			want:  http.StatusOK,
		},
		{
			name:   "without token",
			noAuth: true,
			want:   http.StatusUnauthorized,
		},
		{
			name:      "malformed token",
			malformed: true,
			want:      http.StatusUnauthorized,
		},
		{
			name:  "expired token",
			token: testJWT(testUserID, []string{tenantA}, time.Now().Add(-time.Minute), testJWTIssuer, testJWTAud, testJWTSecret),
			want:  http.StatusUnauthorized,
		},
		{
			name:  "invalid signature",
			token: testJWT(testUserID, []string{tenantA}, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, "wrong-secret"),
			want:  http.StatusUnauthorized,
		},
		{
			name:  "invalid issuer",
			token: testJWT(testUserID, []string{tenantA}, time.Now().Add(time.Hour), "wrong-issuer", testJWTAud, testJWTSecret),
			want:  http.StatusUnauthorized,
		},
		{
			name:  "invalid audience",
			token: testJWT(testUserID, []string{tenantA}, time.Now().Add(time.Hour), testJWTIssuer, "wrong-audience", testJWTSecret),
			want:  http.StatusUnauthorized,
		},
		{
			name:  "alg none rejected",
			token: encodeJWTPart(map[string]any{"alg": "none", "typ": "JWT"}) + "." + encodeJWTPart(map[string]any{"sub": testUserID, "tenant_id": tenantA, "exp": time.Now().Add(time.Hour).Unix(), "iss": testJWTIssuer, "aud": testJWTAud}) + ".",
			want:  http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			switch {
			case tt.noAuth:
			case tt.malformed:
				req.Header.Set("Authorization", "Bearer not-a-jwt")
			default:
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rr := httptest.NewRecorder()
			app.routes().ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("expected %d, got %d: %s", tt.want, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestJWTTenantAuthorization(t *testing.T) {
	tenantA := "550e8400-e29b-41d4-a716-446655440000"
	tenantB := "550e8400-e29b-41d4-a716-446655440099"
	tenantC := "550e8400-e29b-41d4-a716-446655440088"
	app := authenticatedTestApp(&App{CustomerSvc: service.NewCustomerService(&apiCustomerRepo{})})

	tests := []struct {
		name       string
		pathTenant string
		claims     []string
		want       int
	}{
		{name: "tenant A accesses tenant A", pathTenant: tenantA, claims: []string{tenantA}, want: http.StatusOK},
		{name: "tenant A denied tenant B", pathTenant: tenantB, claims: []string{tenantA}, want: http.StatusForbidden},
		{name: "multi tenant allows tenant B", pathTenant: tenantB, claims: []string{tenantA, tenantB}, want: http.StatusOK},
		{name: "multi tenant denies absent tenant", pathTenant: tenantC, claims: []string{tenantA, tenantB}, want: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+tt.pathTenant+"/customers", nil)
			req.Header.Set("Authorization", "Bearer "+testJWT(testUserID, tt.claims, time.Now().Add(time.Hour), testJWTIssuer, testJWTAud, testJWTSecret))
			rr := httptest.NewRecorder()
			app.routes().ServeHTTP(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("expected %d, got %d: %s", tt.want, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestDevelopmentHeadersAreEnvironmentScoped(t *testing.T) {
	tenantID := "550e8400-e29b-41d4-a716-446655440000"
	path := "/api/v1/tenants/" + tenantID + "/customers"

	devReq := httptest.NewRequest(http.MethodGet, path, nil)
	devReq.Header.Set("X-Dev-User-ID", testUserID)
	devReq.Header.Set("X-Dev-Tenant-ID", tenantID)
	devRR := httptest.NewRecorder()
	developmentTestApp(&App{CustomerSvc: service.NewCustomerService(&apiCustomerRepo{})}).routes().ServeHTTP(devRR, devReq)
	if devRR.Code != http.StatusOK {
		t.Fatalf("development headers expected 200, got %d: %s", devRR.Code, devRR.Body.String())
	}

	prodReq := httptest.NewRequest(http.MethodGet, path, nil)
	prodReq.Header.Set("X-Dev-User-ID", testUserID)
	prodReq.Header.Set("X-Dev-Tenant-ID", tenantID)
	prodRR := httptest.NewRecorder()
	authenticatedTestApp(&App{CustomerSvc: service.NewCustomerService(&apiCustomerRepo{})}).routes().ServeHTTP(prodRR, prodReq)
	if prodRR.Code != http.StatusUnauthorized {
		t.Fatalf("production must ignore dev headers; expected 401, got %d: %s", prodRR.Code, prodRR.Body.String())
	}

	legacyReq := httptest.NewRequest(http.MethodGet, path, nil)
	legacyReq.Header.Set("X-User-ID", testUserID)
	legacyReq.Header.Set("X-Tenant-ID", tenantID)
	legacyRR := httptest.NewRecorder()
	authenticatedTestApp(&App{CustomerSvc: service.NewCustomerService(&apiCustomerRepo{})}).routes().ServeHTTP(legacyRR, legacyReq)
	if legacyRR.Code != http.StatusUnauthorized {
		t.Fatalf("production must ignore legacy headers; expected 401, got %d: %s", legacyRR.Code, legacyRR.Body.String())
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
	app := authenticatedTestApp(&App{
		ProviderSvc: service.NewProviderService(providerRepo),
		Factory:     providerFactory,
	})

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
