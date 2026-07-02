package service

import (
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type tenantRepo interface {
	Create(*domain.Tenant) error
	FindByID(string) (*domain.Tenant, error)
	List() ([]domain.Tenant, error)
	Update(*domain.Tenant) error
	Delete(string) error
}

type TenantService struct {
	repo tenantRepo
}

func NewTenantService(repo tenantRepo) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) Create(t *domain.Tenant) error {
	if strings.TrimSpace(t.Name) == "" {
		return ErrValidation
	}
	return s.repo.Create(t)
}

func (s *TenantService) Get(id string) (*domain.Tenant, error) {
	if !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.FindByID(id)
}

func (s *TenantService) List() ([]domain.Tenant, error) {
	return s.repo.List()
}
