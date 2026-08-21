package service

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type onboardingRepo interface {
	CreateTenantOnboarding(domain.OnboardingInput) (*domain.OnboardingResult, error)
}

type OnboardingService struct {
	repo onboardingRepo
}

func NewOnboardingService(repo onboardingRepo) *OnboardingService {
	return &OnboardingService{repo: repo}
}

func (s *OnboardingService) CreateTenant(input domain.OnboardingInput) (*domain.OnboardingResult, error) {
	if err := normalizeAndValidateTenant(&input.Tenant); err != nil {
		return nil, ErrValidation
	}
	if input.Admin != nil {
		input.Admin.Email = NormalizeEmail(input.Admin.Email)
		input.Admin.Name = strings.TrimSpace(input.Admin.Name)
		if !IsValidEmail(input.Admin.Email) {
			return nil, ErrValidation
		}
		if strings.TrimSpace(input.Admin.PasswordHash) == "" {
			return nil, ErrValidation
		}
	}
	for i := range input.Providers {
		input.Providers[i].ProviderID = strings.TrimSpace(input.Providers[i].ProviderID)
		if !IsValidUUID(input.Providers[i].ProviderID) {
			return nil, ErrValidation
		}
		input.Providers[i].Config = NormalizeOptionalString(input.Providers[i].Config)
	}
	result, err := s.repo.CreateTenantOnboarding(input)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProviderNotAllowed
	}
	return result, err
}
