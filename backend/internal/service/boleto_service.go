package service

import (
	"strings"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type boletoRepo interface {
	Create(*domain.Boleto) error
	FindByID(string) (*domain.Boleto, error)
	ListByTenant(string) ([]domain.Boleto, error)
	Update(*domain.Boleto) error
	Delete(string, string) error
}

type BoletoService struct {
	repo boletoRepo
}

func NewBoletoService(repo boletoRepo) *BoletoService {
	return &BoletoService{repo: repo}
}

func (s *BoletoService) Create(b *domain.Boleto) error {
	if !IsValidUUID(b.TenantID) || !IsValidUUID(b.CustomerID) {
		return ErrValidation
	}
	if b.ProviderID != nil && !IsValidUUID(*b.ProviderID) {
		return ErrValidation
	}
	if b.AmountCents <= 0 {
		return ErrValidation
	}
	if b.DueDate.IsZero() {
		return ErrValidation
	}

	b.Status = strings.ToUpper(strings.TrimSpace(b.Status))
	if b.Status == "" {
		b.Status = "CREATED"
	}
	if b.Status != "CREATED" && b.Status != "PENDING" {
		return ErrValidation
	}

	return s.repo.Create(b)
}

func (s *BoletoService) Get(id string) (*domain.Boleto, error) {
	if !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.FindByID(id)
}

func (s *BoletoService) ListByTenant(tenantID string) ([]domain.Boleto, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	return s.repo.ListByTenant(tenantID)
}

func NormalizeDueDate(dateOnly string) (time.Time, error) {
	return time.Parse("2006-01-02", dateOnly)
}
