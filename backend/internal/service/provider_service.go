package service

import (
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type providerRepo interface {
	Create(*domain.Provider) error
	FindByID(string) (*domain.Provider, error)
	ListByTenant(string) ([]domain.Provider, error)
	Update(*domain.Provider) error
	Delete(string, string) error
}

type ProviderService struct {
	repo providerRepo
}

func NewProviderService(repo providerRepo) *ProviderService {
	return &ProviderService{repo: repo}
}

func (s *ProviderService) Create(p *domain.Provider) error {
	if !IsValidUUID(p.TenantID) {
		return ErrValidation
	}
	if strings.TrimSpace(p.Name) == "" {
		return ErrValidation
	}
	if p.Status == "" {
		p.Status = "ACTIVE"
	}
	return s.repo.Create(p)
}

func (s *ProviderService) Get(id string) (*domain.Provider, error) {
	if !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.FindByID(id)
}

func (s *ProviderService) ListByTenant(tenantID string) ([]domain.Provider, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	return s.repo.ListByTenant(tenantID)
}
