package service

import (
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type providerRepo interface {
	Create(*domain.Provider) error
	FindByID(string) (*domain.Provider, error)
	ListByTenant(string) ([]domain.Provider, error)
	ListCatalog() ([]domain.Provider, error)
	FindTenantProvider(string, string) (*domain.TenantProviderConfig, error)
	Update(*domain.Provider) error
	Delete(string, string) error
	SetStatus(string, string) error
	AssignToTenant(string, string, bool, *string) (*domain.TenantProvider, error)
	IsAllowedForTenant(string, string) (bool, error)
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
	p.Name = strings.TrimSpace(p.Name)
	if strings.TrimSpace(p.Name) == "" {
		return ErrValidation
	}
	if p.Status == "" {
		p.Status = "ACTIVE"
	}
	return s.repo.Create(p)
}

func (s *ProviderService) CreateCatalog(p *domain.Provider) error {
	p.TenantID = ""
	p.Name = strings.TrimSpace(p.Name)
	p.Type = strings.TrimSpace(p.Type)
	if strings.TrimSpace(p.Name) == "" {
		return ErrValidation
	}
	if p.Status == "" {
		p.Status = "ACTIVE"
	}
	return s.repo.Create(p)
}

func (s *ProviderService) UpdateCatalog(p *domain.Provider) error {
	if !IsValidUUID(p.ID) {
		return ErrValidation
	}
	p.TenantID = ""
	p.Name = strings.TrimSpace(p.Name)
	p.Type = strings.TrimSpace(p.Type)
	if p.Name == "" {
		return ErrValidation
	}
	if p.Status == "" {
		current, err := s.repo.FindByID(p.ID)
		if err != nil {
			return err
		}
		p.Status = current.Status
	}
	return s.repo.Update(p)
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

func (s *ProviderService) ListCatalog() ([]domain.Provider, error) {
	return s.repo.ListCatalog()
}

func (s *ProviderService) GetTenantProvider(tenantID, providerID string) (*domain.TenantProviderConfig, error) {
	if !IsValidUUID(tenantID) || !IsValidUUID(providerID) {
		return nil, ErrValidation
	}
	cfg, err := s.repo.FindTenantProvider(tenantID, providerID)
	if err != nil {
		return nil, err
	}
	if cfg.Provider.Status != "ACTIVE" || cfg.TenantProvider.DeletedAt != nil || !cfg.TenantProvider.Active {
		return nil, ErrProviderNotAllowed
	}
	return cfg, nil
}

func (s *ProviderService) ActivateCatalog(id string) error {
	return s.setCatalogStatus(id, "ACTIVE")
}

func (s *ProviderService) DeactivateCatalog(id string) error {
	return s.setCatalogStatus(id, "INACTIVE")
}

func (s *ProviderService) setCatalogStatus(id, status string) error {
	if !IsValidUUID(id) {
		return ErrValidation
	}
	return s.repo.SetStatus(id, status)
}

func (s *ProviderService) AssignToTenant(tenantID, providerID string, active bool, config *string) (*domain.TenantProvider, error) {
	if !IsValidUUID(tenantID) || !IsValidUUID(providerID) {
		return nil, ErrValidation
	}
	provider, err := s.repo.FindByID(providerID)
	if err != nil {
		return nil, err
	}
	if provider.Status != "ACTIVE" || provider.TenantID != "" {
		return nil, ErrProviderNotAllowed
	}
	return s.repo.AssignToTenant(tenantID, providerID, active, NormalizeOptionalString(config))
}

func (s *ProviderService) IsAllowedForTenant(tenantID, providerID string) (bool, error) {
	if !IsValidUUID(tenantID) || !IsValidUUID(providerID) {
		return false, ErrValidation
	}
	return s.repo.IsAllowedForTenant(tenantID, providerID)
}
