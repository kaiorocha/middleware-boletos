package service

import (
	"testing"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type tenantRepoMock struct{ created bool }

func (m *tenantRepoMock) Create(*domain.Tenant) error                    { m.created = true; return nil }
func (m *tenantRepoMock) FindByID(string) (*domain.Tenant, error)        { return &domain.Tenant{}, nil }
func (m *tenantRepoMock) List() ([]domain.Tenant, error)                 { return nil, nil }
func (m *tenantRepoMock) Update(*domain.Tenant) error                    { return nil }
func (m *tenantRepoMock) Delete(string) error                            { return nil }

type userRepoMock struct{ created bool }

func (m *userRepoMock) Create(*domain.User) error                        { m.created = true; return nil }
func (m *userRepoMock) FindByID(string) (*domain.User, error)            { return &domain.User{}, nil }
func (m *userRepoMock) ListByTenant(string) ([]domain.User, error)       { return nil, nil }
func (m *userRepoMock) Update(*domain.User) error                        { return nil }
func (m *userRepoMock) Delete(string, string) error                      { return nil }

type customerRepoMock struct{ created bool }

func (m *customerRepoMock) Create(*domain.Customer) error                { m.created = true; return nil }
func (m *customerRepoMock) FindByID(string) (*domain.Customer, error)    { return &domain.Customer{}, nil }
func (m *customerRepoMock) ListByTenant(string) ([]domain.Customer, error) { return nil, nil }
func (m *customerRepoMock) Update(*domain.Customer) error                { return nil }
func (m *customerRepoMock) Delete(string, string) error                  { return nil }

type providerRepoMock struct{ created bool }

func (m *providerRepoMock) Create(*domain.Provider) error                { m.created = true; return nil }
func (m *providerRepoMock) FindByID(string) (*domain.Provider, error)    { return &domain.Provider{}, nil }
func (m *providerRepoMock) ListByTenant(string) ([]domain.Provider, error) { return nil, nil }
func (m *providerRepoMock) Update(*domain.Provider) error                { return nil }
func (m *providerRepoMock) Delete(string, string) error                  { return nil }

type boletoRepoMock struct{ created bool }

func (m *boletoRepoMock) Create(*domain.Boleto) error                    { m.created = true; return nil }
func (m *boletoRepoMock) FindByID(string) (*domain.Boleto, error)        { return &domain.Boleto{}, nil }
func (m *boletoRepoMock) ListByTenant(string) ([]domain.Boleto, error)   { return nil, nil }
func (m *boletoRepoMock) Update(*domain.Boleto) error                    { return nil }
func (m *boletoRepoMock) Delete(string, string) error                    { return nil }

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
