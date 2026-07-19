package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/contracts"
	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/factory"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

type tenantRepoMock struct{ created bool }

func (m *tenantRepoMock) Create(*domain.Tenant) error             { m.created = true; return nil }
func (m *tenantRepoMock) FindByID(string) (*domain.Tenant, error) { return &domain.Tenant{}, nil }
func (m *tenantRepoMock) List() ([]domain.Tenant, error)          { return nil, nil }
func (m *tenantRepoMock) Update(*domain.Tenant) error             { return nil }
func (m *tenantRepoMock) Delete(string) error                     { return nil }

type userRepoMock struct {
	created bool
	err     error
	last    *domain.User
}

func (m *userRepoMock) Create(u *domain.User) error                { m.created = true; m.last = u; return m.err }
func (m *userRepoMock) FindByID(string) (*domain.User, error)      { return &domain.User{}, nil }
func (m *userRepoMock) ListByTenant(string) ([]domain.User, error) { return nil, nil }
func (m *userRepoMock) Update(*domain.User) error                  { return nil }
func (m *userRepoMock) Delete(string, string) error                { return nil }

type customerRepoMock struct {
	created bool
	err     error
	last    *domain.Customer
	found   *domain.Customer
}

func (m *customerRepoMock) Create(c *domain.Customer) error {
	m.created = true
	m.last = c
	return m.err
}
func (m *customerRepoMock) FindByID(string) (*domain.Customer, error) {
	if m.found != nil {
		return m.found, m.err
	}
	return &domain.Customer{}, m.err
}
func (m *customerRepoMock) ListByTenant(string) ([]domain.Customer, error) { return nil, nil }
func (m *customerRepoMock) Update(c *domain.Customer) error                { m.last = c; return m.err }
func (m *customerRepoMock) Delete(string, string) error                    { return nil }

type providerRepoMock struct {
	created bool
	err     error
	last    *domain.Provider
	found   *domain.Provider
}

func (m *providerRepoMock) Create(p *domain.Provider) error {
	m.created = true
	m.last = p
	return m.err
}
func (m *providerRepoMock) FindByID(string) (*domain.Provider, error) {
	if m.found != nil {
		return m.found, m.err
	}
	return &domain.Provider{}, m.err
}
func (m *providerRepoMock) ListByTenant(string) ([]domain.Provider, error) { return nil, nil }
func (m *providerRepoMock) Update(p *domain.Provider) error                { m.last = p; return m.err }
func (m *providerRepoMock) Delete(string, string) error                    { return nil }

type boletoRepoMock struct {
	created bool
	err     error
	last    *domain.Boleto
	found   *domain.Boleto
	updates int
}

func (m *boletoRepoMock) Create(b *domain.Boleto) error { m.created = true; m.last = b; return m.err }
func (m *boletoRepoMock) FindByID(string) (*domain.Boleto, error) {
	if m.found != nil {
		return m.found, m.err
	}
	return &domain.Boleto{}, m.err
}
func (m *boletoRepoMock) ListByTenant(string) ([]domain.Boleto, error) { return nil, nil }
func (m *boletoRepoMock) Update(b *domain.Boleto) error {
	m.updates++
	m.last = b
	return m.err
}
func (m *boletoRepoMock) Delete(string, string) error { return nil }

type blacklistRepoMock struct {
	created bool
	err     error
	last    *domain.BlacklistEntry
	found   *domain.BlacklistEntry
	blocked bool
}

func (m *blacklistRepoMock) Create(entry *domain.BlacklistEntry) error {
	m.created = true
	m.last = entry
	return m.err
}
func (m *blacklistRepoMock) FindByID(string, string) (*domain.BlacklistEntry, error) {
	if m.found != nil {
		return m.found, m.err
	}
	return &domain.BlacklistEntry{}, m.err
}
func (m *blacklistRepoMock) FindByDocument(string, string) (*domain.BlacklistEntry, error) {
	if m.found != nil {
		return m.found, m.err
	}
	return &domain.BlacklistEntry{}, m.err
}
func (m *blacklistRepoMock) List(string, string, *bool) ([]domain.BlacklistEntry, error) {
	if m.found != nil {
		return []domain.BlacklistEntry{*m.found}, m.err
	}
	return nil, m.err
}
func (m *blacklistRepoMock) Update(entry *domain.BlacklistEntry) error {
	m.last = entry
	return m.err
}
func (m *blacklistRepoMock) SoftDelete(string, string) error { return m.err }
func (m *blacklistRepoMock) IsBlocked(string, string) (*domain.BlacklistEntry, bool, error) {
	if m.found != nil {
		return m.found, m.blocked, m.err
	}
	return nil, m.blocked, m.err
}

type blacklistComplianceMock struct {
	entry    *domain.BlacklistEntry
	blocked  bool
	err      error
	attempts int
}

func (m *blacklistComplianceMock) IsBlocked(string, string) (*domain.BlacklistEntry, bool, error) {
	return m.entry, m.blocked, m.err
}
func (m *blacklistComplianceMock) RecordBlockedEmissionAttempt(string, *domain.BlacklistEntry, *domain.Boleto) {
	m.attempts++
}

type providerFactorySpy struct {
	builds  int
	adapter *providerAdapterSpy
	err     error
}

func (f *providerFactorySpy) Build(types.ProviderConfig) (contracts.ProviderAdapter, error) {
	f.builds++
	if f.adapter == nil {
		f.adapter = &providerAdapterSpy{}
	}
	return f.adapter, f.err
}

type providerAdapterSpy struct {
	issues int
}

func (a *providerAdapterSpy) IssueBoleto(context.Context, types.IssueRequest) (types.IssueResponse, error) {
	a.issues++
	return types.IssueResponse{
		ExternalID:    "spy-ext",
		Barcode:       "spy-barcode",
		DigitableLine: "spy-line",
		OurNumber:     "spy-our",
		Status:        types.StatusIssued,
		IssuedAt:      time.Now(),
	}, nil
}
func (a *providerAdapterSpy) GetBoleto(context.Context, types.GetRequest) (types.BoletoSummary, error) {
	return types.BoletoSummary{}, nil
}
func (a *providerAdapterSpy) ListBoletos(context.Context, types.ListRequest) ([]types.BoletoSummary, error) {
	return nil, nil
}
func (a *providerAdapterSpy) CancelBoleto(context.Context, types.CancelRequest) (types.BoletoSummary, error) {
	return types.BoletoSummary{}, nil
}
func (a *providerAdapterSpy) RegisterWebhook(context.Context, types.RegisterWebhookRequest) error {
	return nil
}
func (a *providerAdapterSpy) ValidateWebhook(context.Context, types.ValidateWebhookRequest) (types.WebhookEvent, error) {
	return types.WebhookEvent{}, nil
}
func (a *providerAdapterSpy) GetBalance(context.Context, types.BalanceRequest) (types.BalanceResponse, error) {
	return types.BalanceResponse{}, nil
}
func (a *providerAdapterSpy) Health(context.Context) (types.HealthResponse, error) {
	return types.HealthResponse{}, nil
}

// ========== TenantService Tests ==========

func TestTenantServiceRejectEmptyName(t *testing.T) {
	repo := &tenantRepoMock{}
	svc := NewTenantService(repo)
	err := svc.Create(&domain.Tenant{Name: ""})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestTenantServiceCreateValid(t *testing.T) {
	repo := &tenantRepoMock{}
	svc := NewTenantService(repo)
	err := svc.Create(&domain.Tenant{Name: "acme-corp"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected tenant to be created")
	}
}

// ========== UserService Tests ==========

func TestUserServiceRejectInvalidTenantID(t *testing.T) {
	repo := &userRepoMock{}
	svc := NewUserService(repo)
	err := svc.Create(&domain.User{
		TenantID: "not-a-uuid",
		Email:    "valid@example.com",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid tenant_id")
	}
}

func TestUserServiceRejectInvalidEmail(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &userRepoMock{}
	svc := NewUserService(repo)
	err := svc.Create(&domain.User{
		TenantID: validUUID,
		Email:    "invalid-email",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestUserServiceRejectEmptyEmail(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &userRepoMock{}
	svc := NewUserService(repo)
	err := svc.Create(&domain.User{
		TenantID: validUUID,
		Email:    "",
	})
	if err == nil {
		t.Fatal("expected validation error for empty email")
	}
}

func TestUserServiceCreateValid(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &userRepoMock{}
	svc := NewUserService(repo)
	err := svc.Create(&domain.User{
		TenantID: validUUID,
		Email:    "user@example.com",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected user to be created")
	}
}

func TestUserServiceNormalizesEmail(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &userRepoMock{}
	svc := NewUserService(repo)
	err := svc.Create(&domain.User{
		TenantID: validUUID,
		Email:    "  TESTE@EMAIL.COM  ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.last.Email != "teste@email.com" {
		t.Fatalf("expected normalized email, got %q", repo.last.Email)
	}
}

func TestUserServicePropagatesDuplicateError(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &userRepoMock{err: NewDuplicateResource("duplicated")}
	svc := NewUserService(repo)
	err := svc.Create(&domain.User{
		TenantID: validUUID,
		Email:    "user@example.com",
	})
	if !errors.Is(err, ErrDuplicateResource) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

// ========== CustomerService Tests ==========

func TestCustomerServiceRejectInvalidTenantID(t *testing.T) {
	repo := &customerRepoMock{}
	svc := NewCustomerService(repo)
	err := svc.Create(&domain.Customer{
		TenantID: "not-a-uuid",
		Name:     "Customer Name",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid tenant_id")
	}
}

func TestCustomerServiceRejectEmptyName(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &customerRepoMock{}
	svc := NewCustomerService(repo)
	err := svc.Create(&domain.Customer{
		TenantID: validUUID,
		Name:     "",
	})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestCustomerServiceCreateValid(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &customerRepoMock{}
	svc := NewCustomerService(repo)
	err := svc.Create(&domain.Customer{
		TenantID: validUUID,
		Name:     "ACME Company",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected customer to be created")
	}
}

func TestCustomerServiceNormalizesDocument(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	document := "123.456.789-00"
	repo := &customerRepoMock{}
	svc := NewCustomerService(repo)
	err := svc.Create(&domain.Customer{
		TenantID: validUUID,
		Name:     "Customer Name",
		Document: &document,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.last.Document == nil || *repo.last.Document != "12345678900" {
		t.Fatalf("expected normalized document, got %v", repo.last.Document)
	}
}

func TestCustomerServiceEmptyDocumentBecomesNil(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	document := " - ./ "
	repo := &customerRepoMock{}
	svc := NewCustomerService(repo)
	err := svc.Create(&domain.Customer{
		TenantID: validUUID,
		Name:     "Customer Name",
		Document: &document,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.last.Document != nil {
		t.Fatalf("expected nil document, got %v", repo.last.Document)
	}
}

func TestCustomerServicePropagatesDuplicateError(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &customerRepoMock{err: NewDuplicateResource("duplicated")}
	svc := NewCustomerService(repo)
	err := svc.Create(&domain.Customer{
		TenantID: validUUID,
		Name:     "Customer Name",
	})
	if !errors.Is(err, ErrDuplicateResource) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

// ========== ProviderService Tests ==========

func TestProviderServiceRejectInvalidTenantID(t *testing.T) {
	repo := &providerRepoMock{}
	svc := NewProviderService(repo)
	err := svc.Create(&domain.Provider{
		TenantID: "not-a-uuid",
		Name:     "Provider Name",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid tenant_id")
	}
}

func TestProviderServiceRejectEmptyName(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &providerRepoMock{}
	svc := NewProviderService(repo)
	err := svc.Create(&domain.Provider{
		TenantID: validUUID,
		Name:     "",
	})
	if err == nil {
		t.Fatal("expected validation error for empty name")
	}
}

func TestProviderServiceCreateValid(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &providerRepoMock{}
	svc := NewProviderService(repo)
	err := svc.Create(&domain.Provider{
		TenantID: validUUID,
		Name:     "Banco do Brasil",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected provider to be created")
	}
}

func TestProviderServiceNormalizesName(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &providerRepoMock{}
	svc := NewProviderService(repo)
	err := svc.Create(&domain.Provider{
		TenantID: validUUID,
		Name:     "  Banco Demo  ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.last.Name != "Banco Demo" {
		t.Fatalf("expected trimmed provider name, got %q", repo.last.Name)
	}
}

func TestProviderServicePropagatesDuplicateError(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &providerRepoMock{err: NewDuplicateResource("duplicated")}
	svc := NewProviderService(repo)
	err := svc.Create(&domain.Provider{
		TenantID: validUUID,
		Name:     "Banco Demo",
	})
	if !errors.Is(err, ErrDuplicateResource) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

// ========== BoletoService Tests ==========

func TestBoletoServiceRejectInvalidTenantID(t *testing.T) {
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validDueDate := time.Now().AddDate(0, 0, 7)
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    "not-a-uuid",
		CustomerID:  validCustomerUUID,
		AmountCents: 10000,
		DueDate:     validDueDate,
		Status:      "CREATED",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid tenant_id")
	}
}

func TestBoletoServiceRejectInvalidCustomerID(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validDueDate := time.Now().AddDate(0, 0, 7)
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  "not-a-uuid",
		AmountCents: 10000,
		DueDate:     validDueDate,
		Status:      "CREATED",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid customer_id")
	}
}

func TestBoletoServiceRejectInvalidProviderID(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validDueDate := time.Now().AddDate(0, 0, 7)
	invalidProviderID := "not-a-uuid"
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		ProviderID:  &invalidProviderID,
		AmountCents: 10000,
		DueDate:     validDueDate,
		Status:      "CREATED",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid provider_id")
	}
}

func TestBoletoServiceRejectZeroAmount(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validDueDate := time.Now().AddDate(0, 0, 7)
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: 0,
		DueDate:     validDueDate,
		Status:      "CREATED",
	})
	if err == nil {
		t.Fatal("expected validation error for zero amount")
	}
}

func TestBoletoServiceRejectNegativeAmount(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validDueDate := time.Now().AddDate(0, 0, 7)
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: -1000,
		DueDate:     validDueDate,
		Status:      "CREATED",
	})
	if err == nil {
		t.Fatal("expected validation error for negative amount")
	}
}

func TestBoletoServiceRejectEmptyDueDate(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: 10000,
		DueDate:     time.Time{},
		Status:      "CREATED",
	})
	if err == nil {
		t.Fatal("expected validation error for empty due_date")
	}
}

func TestBoletoServiceRejectInvalidStatus(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validDueDate := time.Now().AddDate(0, 0, 7)
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: 10000,
		DueDate:     validDueDate,
		Status:      "INVALID_STATUS",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid status")
	}
}

func TestBoletoServiceCreateValidWithCREATED(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validDueDate := time.Now().AddDate(0, 0, 7)
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: 50000,
		DueDate:     validDueDate,
		Status:      "CREATED",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected boleto to be created")
	}
}

func TestBoletoServiceCreateValidWithPROCESSING(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validDueDate := time.Now().AddDate(0, 0, 7)
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: 75000,
		DueDate:     validDueDate,
		Status:      "PROCESSING",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected boleto to be created")
	}
}

func TestBoletoServiceEmitUsesProviderAdapter(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"
	providerName := "Mock"

	boletoRepo := &boletoRepoMock{found: &domain.Boleto{
		ID:          boletoID,
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		ProviderID:  &validProviderUUID,
		AmountCents: 25000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
	}}
	providerRepo := &providerRepoMock{found: &domain.Provider{
		ID:       validProviderUUID,
		TenantID: validTenantUUID,
		Name:     providerName,
		Status:   "ACTIVE",
	}}

	svc := NewBoletoService(boletoRepo).
		WithCustomerRepository(&customerRepoMock{found: completeCustomer(validTenantUUID)}).
		WithProviderRepository(providerRepo).
		WithBlacklistService(&blacklistComplianceMock{}).
		WithProviderFactory(factory.NewProviderFactory())

	got, err := svc.Emit(context.Background(), validTenantUUID, boletoID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Status != "ISSUED" {
		t.Fatalf("expected ISSUED, got %q", got.Status)
	}
	if got.ExternalID == nil || got.Barcode == nil || got.DigitableLine == nil || got.OurNumber == nil || got.IssuedAt == nil {
		t.Fatalf("expected provider fields to be persisted: %+v", got)
	}
	if boletoRepo.updates != 2 {
		t.Fatalf("expected processing and issued updates, got %d", boletoRepo.updates)
	}
}

func TestBoletoServiceEmitBlocksBlacklistedCustomerBeforeProvider(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"

	boletoRepo := &boletoRepoMock{found: &domain.Boleto{
		ID:          boletoID,
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		ProviderID:  &validProviderUUID,
		AmountCents: 25000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
	}}
	blacklist := &blacklistComplianceMock{
		blocked: true,
		entry: &domain.BlacklistEntry{
			ID:       "550e8400-e29b-41d4-a716-446655440004",
			TenantID: validTenantUUID,
			Document: "12345678900",
			Reason:   "Solicitação do cliente",
			Active:   true,
		},
	}
	svc := NewBoletoService(boletoRepo).
		WithCustomerRepository(&customerRepoMock{found: completeCustomer(validTenantUUID)}).
		WithProviderRepository(&providerRepoMock{}).
		WithBlacklistService(blacklist).
		WithProviderFactory(factory.NewProviderFactory())

	_, err := svc.Emit(context.Background(), validTenantUUID, boletoID)
	if !errors.Is(err, ErrCustomerBlocked) {
		t.Fatalf("expected customer blocked error, got %v", err)
	}
	if boletoRepo.updates != 0 {
		t.Fatalf("expected no boleto updates before provider flow, got %d", boletoRepo.updates)
	}
	if blacklist.attempts != 1 {
		t.Fatalf("expected blocked attempt audit, got %d", blacklist.attempts)
	}
}

func TestBoletoServiceEmitFailsClosedWithoutBlacklistService(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"
	spy := &providerFactorySpy{adapter: &providerAdapterSpy{}}

	svc := NewBoletoService(&boletoRepoMock{found: &domain.Boleto{
		ID: boletoID, TenantID: validTenantUUID, CustomerID: validCustomerUUID, ProviderID: &validProviderUUID, AmountCents: 25000, DueDate: time.Now().AddDate(0, 0, 7), Status: "CREATED",
	}}).
		WithCustomerRepository(&customerRepoMock{found: completeCustomer(validTenantUUID)}).
		WithProviderRepository(&providerRepoMock{}).
		WithProviderFactory(spy)

	_, err := svc.Emit(context.Background(), validTenantUUID, boletoID)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if spy.builds != 0 || spy.adapter.issues != 0 {
		t.Fatalf("expected no provider interaction, builds=%d issues=%d", spy.builds, spy.adapter.issues)
	}
}

func TestBoletoServiceEmitRejectsCustomerWithoutDocument(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"
	customer := completeCustomer(validTenantUUID)
	customer.Document = nil
	spy := &providerFactorySpy{adapter: &providerAdapterSpy{}}
	blacklist := &blacklistComplianceMock{}

	svc := NewBoletoService(&boletoRepoMock{found: &domain.Boleto{
		ID: boletoID, TenantID: validTenantUUID, CustomerID: validCustomerUUID, ProviderID: &validProviderUUID, AmountCents: 25000, DueDate: time.Now().AddDate(0, 0, 7), Status: "CREATED",
	}}).
		WithCustomerRepository(&customerRepoMock{found: customer}).
		WithProviderRepository(&providerRepoMock{}).
		WithBlacklistService(blacklist).
		WithProviderFactory(spy)

	_, err := svc.Emit(context.Background(), validTenantUUID, boletoID)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if spy.builds != 0 || spy.adapter.issues != 0 || blacklist.attempts != 0 {
		t.Fatalf("expected no provider/bypass interaction, builds=%d issues=%d attempts=%d", spy.builds, spy.adapter.issues, blacklist.attempts)
	}
}

func TestBoletoServiceEmitStopsWhenBlacklistServiceErrors(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"
	spy := &providerFactorySpy{adapter: &providerAdapterSpy{}}
	blacklistErr := errors.New("blacklist unavailable")

	svc := NewBoletoService(&boletoRepoMock{found: &domain.Boleto{
		ID: boletoID, TenantID: validTenantUUID, CustomerID: validCustomerUUID, ProviderID: &validProviderUUID, AmountCents: 25000, DueDate: time.Now().AddDate(0, 0, 7), Status: "CREATED",
	}}).
		WithCustomerRepository(&customerRepoMock{found: completeCustomer(validTenantUUID)}).
		WithProviderRepository(&providerRepoMock{}).
		WithBlacklistService(&blacklistComplianceMock{err: blacklistErr}).
		WithProviderFactory(spy)

	_, err := svc.Emit(context.Background(), validTenantUUID, boletoID)
	if !errors.Is(err, blacklistErr) {
		t.Fatalf("expected blacklist error, got %v", err)
	}
	if spy.builds != 0 || spy.adapter.issues != 0 {
		t.Fatalf("expected no provider interaction, builds=%d issues=%d", spy.builds, spy.adapter.issues)
	}
}

func TestBoletoServiceEmitAllowedCustomerCallsProvider(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"
	providerRepo := &providerRepoMock{found: &domain.Provider{ID: validProviderUUID, TenantID: validTenantUUID, Name: "Mock", Status: "ACTIVE"}}
	adapter := &providerAdapterSpy{}
	spy := &providerFactorySpy{adapter: adapter}

	got, err := NewBoletoService(&boletoRepoMock{found: &domain.Boleto{
		ID: boletoID, TenantID: validTenantUUID, CustomerID: validCustomerUUID, ProviderID: &validProviderUUID, AmountCents: 25000, DueDate: time.Now().AddDate(0, 0, 7), Status: "CREATED",
	}}).
		WithCustomerRepository(&customerRepoMock{found: completeCustomer(validTenantUUID)}).
		WithProviderRepository(providerRepo).
		WithBlacklistService(&blacklistComplianceMock{}).
		WithProviderFactory(spy).
		Emit(context.Background(), validTenantUUID, boletoID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Status != "ISSUED" || spy.builds != 1 || adapter.issues != 1 {
		t.Fatalf("expected provider call and issued boleto, builds=%d issues=%d boleto=%+v", spy.builds, adapter.issues, got)
	}
}

func TestBoletoServiceEmitIsIdempotentWhenAlreadyIssued(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"
	externalID := "ext-123"
	ourNumber := "our-123"

	boletoRepo := &boletoRepoMock{found: &domain.Boleto{
		ID:          boletoID,
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		ProviderID:  &validProviderUUID,
		AmountCents: 25000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "ISSUED",
		ExternalID:  &externalID,
		OurNumber:   &ourNumber,
	}}

	svc := NewBoletoService(boletoRepo).
		WithCustomerRepository(&customerRepoMock{}).
		WithProviderRepository(&providerRepoMock{}).
		WithBlacklistService(&blacklistComplianceMock{}).
		WithProviderFactory(factory.NewProviderFactory())

	got, err := svc.Emit(context.Background(), validTenantUUID, boletoID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ExternalID == nil || *got.ExternalID != externalID {
		t.Fatalf("expected existing boleto, got %+v", got)
	}
	if boletoRepo.updates != 0 {
		t.Fatalf("expected no provider call/update for idempotent emit, got %d updates", boletoRepo.updates)
	}
}

func TestBoletoServiceEmitReturnsInvalidPayerForIncompleteCustomer(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"

	boletoRepo := &boletoRepoMock{found: &domain.Boleto{
		ID:          boletoID,
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		ProviderID:  &validProviderUUID,
		AmountCents: 25000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
	}}
	providerRepo := &providerRepoMock{found: &domain.Provider{
		ID:       validProviderUUID,
		TenantID: validTenantUUID,
		Name:     "Mock",
		Status:   "ACTIVE",
	}}
	customer := completeCustomer(validTenantUUID)
	customer.PostalCode = nil

	svc := NewBoletoService(boletoRepo).
		WithCustomerRepository(&customerRepoMock{found: customer}).
		WithProviderRepository(providerRepo).
		WithBlacklistService(&blacklistComplianceMock{}).
		WithProviderFactory(factory.NewProviderFactory())

	_, err := svc.Emit(context.Background(), validTenantUUID, boletoID)
	perr, ok := err.(*providererrors.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if perr.Code != "INVALID_PAYER" {
		t.Fatalf("expected INVALID_PAYER, got %s", perr.Code)
	}
}

func TestBoletoServiceEmitMoncalieriWithCompleteCustomer(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	boletoID := "550e8400-e29b-41d4-a716-446655440003"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/CashIn/GerarBoleto" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var payload struct {
			Data struct {
				DadosSacado struct {
					CpfCnpj int64  `json:"CpfCnpj"`
					Cep     int    `json:"Cep"`
					Uf      string `json:"Uf"`
				} `json:"DadosSacado"`
			} `json:"Data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("invalid payload: %v", err)
		}
		if payload.Data.DadosSacado.CpfCnpj != 12345678900 || payload.Data.DadosSacado.Cep != 12345678 || payload.Data.DadosSacado.Uf != "SP" {
			t.Fatalf("unexpected payer payload: %+v", payload.Data.DadosSacado)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Data": map[string]any{
				"NossoNumero":    "NN123",
				"LinhaDigitavel": "linha",
				"CodigoBarras":   "barra",
			},
		})
	}))
	defer server.Close()

	config := `{"base_url":"` + server.URL + `","api_key":"test-key","codigo_canal":1,"codigo_cliente":2}`
	boletoRepo := &boletoRepoMock{found: &domain.Boleto{
		ID:          boletoID,
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		ProviderID:  &validProviderUUID,
		AmountCents: 25000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
	}}
	providerRepo := &providerRepoMock{found: &domain.Provider{
		ID:       validProviderUUID,
		TenantID: validTenantUUID,
		Name:     "Moncalieri Capital",
		Status:   "ACTIVE",
		Config:   &config,
	}}

	svc := NewBoletoService(boletoRepo).
		WithCustomerRepository(&customerRepoMock{found: completeCustomer(validTenantUUID)}).
		WithProviderRepository(providerRepo).
		WithBlacklistService(&blacklistComplianceMock{}).
		WithProviderFactory(factory.NewProviderFactory())

	got, err := svc.Emit(context.Background(), validTenantUUID, boletoID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Status != "ISSUED" || got.OurNumber == nil || *got.OurNumber != "NN123" {
		t.Fatalf("unexpected boleto: %+v", got)
	}
}

func TestBoletoServiceCreateWithValidProvider(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	validProviderUUID := "550e8400-e29b-41d4-a716-446655440002"
	validDueDate := time.Now().AddDate(0, 0, 7)
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		ProviderID:  &validProviderUUID,
		AmountCents: 25000,
		DueDate:     validDueDate,
		Status:      "CREATED",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected boleto to be created")
	}
}

func TestBoletoServiceTrimsExternalIDAndOurNumber(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	externalID := "  ext-123  "
	ourNumber := "  nosso-456  "
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: 25000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
		ExternalID:  &externalID,
		OurNumber:   &ourNumber,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.last.ExternalID == nil || *repo.last.ExternalID != "ext-123" {
		t.Fatalf("expected trimmed external_id, got %v", repo.last.ExternalID)
	}
	if repo.last.OurNumber == nil || *repo.last.OurNumber != "nosso-456" {
		t.Fatalf("expected trimmed our_number, got %v", repo.last.OurNumber)
	}
}

func TestBoletoServiceEmptyExternalIDAndOurNumberBecomeNil(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	externalID := "   "
	ourNumber := "   "
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: 25000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
		ExternalID:  &externalID,
		OurNumber:   &ourNumber,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.last.ExternalID != nil {
		t.Fatalf("expected nil external_id, got %v", repo.last.ExternalID)
	}
	if repo.last.OurNumber != nil {
		t.Fatalf("expected nil our_number, got %v", repo.last.OurNumber)
	}
}

func TestBoletoServicePropagatesDuplicateError(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	validCustomerUUID := "550e8400-e29b-41d4-a716-446655440001"
	repo := &boletoRepoMock{err: NewDuplicateResource("duplicated")}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    validTenantUUID,
		CustomerID:  validCustomerUUID,
		AmountCents: 25000,
		DueDate:     time.Now().AddDate(0, 0, 7),
		Status:      "CREATED",
	})
	if !errors.Is(err, ErrDuplicateResource) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

// ========== BlacklistService Tests ==========

func TestBlacklistServiceCreateNormalizesDocumentAndDefaults(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &blacklistRepoMock{}
	svc := NewBlacklistService(repo)

	err := svc.Create(&domain.BlacklistEntry{
		TenantID: validTenantUUID,
		Document: "123.456.789-00",
		Name:     " Cliente Demo ",
		Reason:   " Solicitação do cliente ",
		Source:   "manual",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected blacklist entry to be created")
	}
	if repo.last.Document != "12345678900" {
		t.Fatalf("expected normalized document, got %q", repo.last.Document)
	}
	if repo.last.Source != "MANUAL" || !repo.last.Active {
		t.Fatalf("expected source MANUAL and active true, got %+v", repo.last)
	}
}

func TestBlacklistServicePropagatesDuplicateError(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &blacklistRepoMock{err: NewDuplicateResource("Este documento já está bloqueado neste tenant.")}
	svc := NewBlacklistService(repo)

	err := svc.Create(&domain.BlacklistEntry{
		TenantID: validTenantUUID,
		Document: "12345678900",
		Reason:   "Solicitação do cliente",
	})
	if !errors.Is(err, ErrDuplicateResource) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestBlacklistServiceIsBlockedNormalizesDocument(t *testing.T) {
	validTenantUUID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &blacklistRepoMock{
		blocked: true,
		found: &domain.BlacklistEntry{
			TenantID: validTenantUUID,
			Document: "12345678900",
			Reason:   "Solicitação do cliente",
			Active:   true,
		},
	}
	svc := NewBlacklistService(repo)

	entry, blocked, err := svc.IsBlocked(validTenantUUID, "123.456.789-00")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !blocked || entry == nil || entry.Document != "12345678900" {
		t.Fatalf("expected blocked normalized document, got blocked=%v entry=%+v", blocked, entry)
	}
}

func completeCustomer(tenantID string) *domain.Customer {
	return &domain.Customer{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		TenantID:   tenantID,
		Name:       "Cliente Demo",
		Document:   testStringPtr("123.456.789-00"),
		Email:      testStringPtr("CLIENTE@EXAMPLE.COM"),
		Address:    testStringPtr("Rua Um"),
		Number:     testStringPtr("123"),
		Complement: testStringPtr("Apto 4"),
		District:   testStringPtr("Centro"),
		City:       testStringPtr("Sao Paulo"),
		State:      testStringPtr("sp"),
		PostalCode: testStringPtr("12345-678"),
		Status:     "ACTIVE",
	}
}

func testStringPtr(value string) *string {
	return &value
}
