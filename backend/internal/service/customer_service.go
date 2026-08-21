package service

import (
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type customerRepo interface {
	Create(*domain.Customer) error
	FindByID(string) (*domain.Customer, error)
	ListByTenant(string) ([]domain.Customer, error)
	Update(*domain.Customer) error
	Delete(string, string) error
}

type CustomerService struct {
	repo customerRepo
}

func NewCustomerService(repo customerRepo) *CustomerService {
	return &CustomerService{repo: repo}
}

func (s *CustomerService) Create(c *domain.Customer) error {
	normalizeCustomer(c)
	if !IsValidUUID(c.TenantID) {
		return ErrValidation
	}
	if strings.TrimSpace(c.Name) == "" {
		return ErrValidation
	}
	if !validateCustomerOptionalFields(c) {
		return ErrValidation
	}
	if c.Status == "" {
		c.Status = "ACTIVE"
	}
	return s.repo.Create(c)
}

func (s *CustomerService) Get(id string) (*domain.Customer, error) {
	if !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.FindByID(id)
}

func (s *CustomerService) ListByTenant(tenantID string) ([]domain.Customer, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	return s.repo.ListByTenant(tenantID)
}

func (s *CustomerService) Update(c *domain.Customer) error {
	normalizeCustomer(c)
	if !IsValidUUID(c.ID) || !IsValidUUID(c.TenantID) {
		return ErrValidation
	}
	if strings.TrimSpace(c.Name) == "" {
		return ErrValidation
	}
	if !validateCustomerOptionalFields(c) {
		return ErrValidation
	}
	if c.Status == "" {
		c.Status = "ACTIVE"
	}
	return s.repo.Update(c)
}

func normalizeCustomer(c *domain.Customer) {
	c.Name = strings.TrimSpace(c.Name)
	c.Document = NormalizeDocument(c.Document)
	c.Email = NormalizeOptionalEmail(c.Email)
	c.Address = NormalizeOptionalString(c.Address)
	c.Number = NormalizeOptionalString(c.Number)
	c.Complement = NormalizeOptionalString(c.Complement)
	c.District = NormalizeOptionalString(c.District)
	c.City = NormalizeOptionalString(c.City)
	c.State = NormalizeOptionalString(c.State)
	if c.State != nil {
		v := strings.ToUpper(*c.State)
		c.State = &v
	}
	c.PostalCode = NormalizePostalCode(c.PostalCode)
	c.ExternalID = NormalizeOptionalString(c.ExternalID)
}

func validateCustomerOptionalFields(c *domain.Customer) bool {
	if c.Email != nil && !IsValidEmail(*c.Email) {
		return false
	}
	if c.State != nil && len(*c.State) != 2 {
		return false
	}
	if c.PostalCode != nil && len(*c.PostalCode) != 8 {
		return false
	}
	if c.Document != nil && len(*c.Document) != 11 && len(*c.Document) != 14 {
		return false
	}
	return true
}
