package service

import (
	"errors"
	"testing"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
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
}

func (m *customerRepoMock) Create(c *domain.Customer) error {
	m.created = true
	m.last = c
	return m.err
}
func (m *customerRepoMock) FindByID(string) (*domain.Customer, error)      { return &domain.Customer{}, nil }
func (m *customerRepoMock) ListByTenant(string) ([]domain.Customer, error) { return nil, nil }
func (m *customerRepoMock) Update(c *domain.Customer) error                { m.last = c; return m.err }
func (m *customerRepoMock) Delete(string, string) error                    { return nil }

type providerRepoMock struct {
	created bool
	err     error
	last    *domain.Provider
}

func (m *providerRepoMock) Create(p *domain.Provider) error {
	m.created = true
	m.last = p
	return m.err
}
func (m *providerRepoMock) FindByID(string) (*domain.Provider, error)      { return &domain.Provider{}, nil }
func (m *providerRepoMock) ListByTenant(string) ([]domain.Provider, error) { return nil, nil }
func (m *providerRepoMock) Update(p *domain.Provider) error                { m.last = p; return m.err }
func (m *providerRepoMock) Delete(string, string) error                    { return nil }

type boletoRepoMock struct {
	created bool
	err     error
	last    *domain.Boleto
}

func (m *boletoRepoMock) Create(b *domain.Boleto) error                { m.created = true; m.last = b; return m.err }
func (m *boletoRepoMock) FindByID(string) (*domain.Boleto, error)      { return &domain.Boleto{}, nil }
func (m *boletoRepoMock) ListByTenant(string) ([]domain.Boleto, error) { return nil, nil }
func (m *boletoRepoMock) Update(b *domain.Boleto) error                { m.last = b; return m.err }
func (m *boletoRepoMock) Delete(string, string) error                  { return nil }

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

func TestBoletoServiceCreateValidWithPENDING(t *testing.T) {
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
		Status:      "PENDING",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.created {
		t.Fatal("expected boleto to be created")
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
