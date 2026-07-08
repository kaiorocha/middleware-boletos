package service

import (
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type userRepo interface {
	Create(*domain.User) error
	FindByID(string) (*domain.User, error)
	ListByTenant(string) ([]domain.User, error)
	Update(*domain.User) error
	Delete(string, string) error
}

type UserService struct {
	repo userRepo
}

func NewUserService(repo userRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(u *domain.User) error {
	u.Email = NormalizeEmail(u.Email)
	if !IsValidUUID(u.TenantID) {
		return ErrValidation
	}
	if !IsValidEmail(u.Email) {
		return ErrValidation
	}
	if strings.TrimSpace(u.Status) == "" {
		u.Status = "ACTIVE"
	}
	return s.repo.Create(u)
}

func (s *UserService) Get(id string) (*domain.User, error) {
	if !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.FindByID(id)
}

func (s *UserService) ListByTenant(tenantID string) ([]domain.User, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	return s.repo.ListByTenant(tenantID)
}
