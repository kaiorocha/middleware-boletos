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

func TestTenantServiceValidation(t *testing.T) {
	repo := &tenantRepoMock{}
	svc := NewTenantService(repo)
	err := svc.Create(&domain.Tenant{Name: ""})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestUserServiceValidation(t *testing.T) {
	repo := &userRepoMock{}
	svc := NewUserService(repo)
	err := svc.Create(&domain.User{TenantID: "invalid", Email: "x"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCustomerServiceValidation(t *testing.T) {
	repo := &customerRepoMock{}
	svc := NewCustomerService(repo)
	err := svc.Create(&domain.Customer{TenantID: "invalid", Name: ""})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProviderServiceValidation(t *testing.T) {
	repo := &providerRepoMock{}
	svc := NewProviderService(repo)
	err := svc.Create(&domain.Provider{TenantID: "invalid", Name: ""})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBoletoServiceValidation(t *testing.T) {
	repo := &boletoRepoMock{}
	svc := NewBoletoService(repo)
	err := svc.Create(&domain.Boleto{
		TenantID:    "invalid",
		CustomerID:  "invalid",
		AmountCents: 0,
		DueDate:     time.Time{},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
