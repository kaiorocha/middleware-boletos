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
	c.Document = NormalizeDocument(c.Document)
	if !IsValidUUID(c.TenantID) {
		return ErrValidation
	}
	c.Name = strings.TrimSpace(c.Name)
	if strings.TrimSpace(c.Name) == "" {
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
	c.Document = NormalizeDocument(c.Document)
	if !IsValidUUID(c.ID) || !IsValidUUID(c.TenantID) {
		return ErrValidation
	}
	c.Name = strings.TrimSpace(c.Name)
	if strings.TrimSpace(c.Name) == "" {
		return ErrValidation
	}
	if c.Status == "" {
		c.Status = "ACTIVE"
	}
	return s.repo.Update(c)
}
