package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type tenantAPITokenRepo interface {
	Rotate(*domain.TenantAPIToken, string) error
	FindActiveByHash(string) (*domain.TenantAPIToken, error)
}

type TenantAPITokenService struct{ repo tenantAPITokenRepo }

func NewTenantAPITokenService(repo tenantAPITokenRepo) *TenantAPITokenService {
	return &TenantAPITokenService{repo: repo}
}

func (s *TenantAPITokenService) Issue(tenantID, environment string) (*domain.TenantAPIToken, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	environment = strings.ToUpper(strings.TrimSpace(environment))
	if environment != "HML" && environment != "PRODUCTION" {
		return nil, ErrValidation
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}
	prefix := "giga_hml_"
	if environment == "PRODUCTION" {
		prefix = "giga_prod_"
	}
	plain := prefix + base64.RawURLEncoding.EncodeToString(random)
	token := &domain.TenantAPIToken{TenantID: tenantID, Environment: environment, TokenPrefix: plain[:min(len(plain), 18)], Token: plain}
	if err := s.repo.Rotate(token, hashTenantAPIToken(plain)); err != nil {
		return nil, err
	}
	return token, nil
}

func (s *TenantAPITokenService) Authenticate(plain string) (*domain.TenantAPIToken, error) {
	plain = strings.TrimSpace(plain)
	if !strings.HasPrefix(plain, "giga_hml_") && !strings.HasPrefix(plain, "giga_prod_") {
		return nil, ErrValidation
	}
	return s.repo.FindActiveByHash(hashTenantAPIToken(plain))
}

func hashTenantAPIToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
