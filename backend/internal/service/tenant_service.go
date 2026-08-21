package service

import (
	"net/url"
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
	if err := normalizeAndValidateTenant(t); err != nil {
		return ErrValidation
	}
	return s.repo.Create(t)
}

func (s *TenantService) Update(t *domain.Tenant) error {
	if !IsValidUUID(t.ID) {
		return ErrValidation
	}
	if err := normalizeAndValidateTenant(t); err != nil {
		return err
	}
	return s.repo.Update(t)
}

func (s *TenantService) Get(id string) (*domain.Tenant, error) {
	if !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.FindByID(id)
}

func normalizeAndValidateTenant(t *domain.Tenant) error {
	t.Name = strings.TrimSpace(t.Name)
	t.Document = normalizeDocumentValue(t.Document)
	t.Address = strings.TrimSpace(t.Address)
	t.District = strings.TrimSpace(t.District)
	t.City = strings.TrimSpace(t.City)
	t.PostalCode = normalizeDocumentValue(t.PostalCode)
	t.State = strings.ToUpper(strings.TrimSpace(t.State))
	t.CountryCode = normalizeDocumentValue(t.CountryCode)
	t.AreaCode = normalizeDocumentValue(t.AreaCode)
	t.PhoneNumber = normalizeDocumentValue(t.PhoneNumber)
	t.WebhookURL = strings.TrimSpace(t.WebhookURL)
	if t.Name == "" || len(t.Document) != 14 || t.Address == "" || t.District == "" || t.City == "" || len(t.PostalCode) != 8 || len(t.State) != 2 || t.CountryCode == "" || t.AreaCode == "" || t.PhoneNumber == "" {
		return ErrValidation
	}
	if t.WebhookURL != "" {
		parsed, err := url.ParseRequestURI(t.WebhookURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return ErrValidation
		}
	}
	return nil
}

func (s *TenantService) List() ([]domain.Tenant, error) {
	return s.repo.List()
}

func (s *TenantService) Delete(id string) error {
	if !IsValidUUID(id) {
		return ErrValidation
	}
	return s.repo.Delete(id)
}
