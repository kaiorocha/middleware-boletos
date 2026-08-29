package service

import (
	"crypto/aes"
	"crypto/cipher"
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

type secureTenantAPITokenRepo interface {
	RotateSecure(*domain.TenantAPIToken, string, string) error
	ListActiveByTenant(string) ([]domain.TenantAPIToken, error)
	FindActiveByTenantEnvironment(string, string) (*domain.TenantAPIToken, error)
}

type TenantAPITokenService struct {
	repo          tenantAPITokenRepo
	encryptionKey []byte
}

func NewTenantAPITokenService(repo tenantAPITokenRepo, secret ...string) *TenantAPITokenService {
	s := &TenantAPITokenService{repo: repo}
	if len(secret) > 0 && strings.TrimSpace(secret[0]) != "" {
		sum := sha256.Sum256([]byte(secret[0]))
		s.encryptionKey = sum[:]
	}
	return s
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
	encrypted, err := s.encrypt(plain)
	if err != nil {
		return nil, err
	}
	if secure, ok := s.repo.(secureTenantAPITokenRepo); ok {
		err = secure.RotateSecure(token, hashTenantAPIToken(plain), encrypted)
	} else {
		err = s.repo.Rotate(token, hashTenantAPIToken(plain))
	}
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *TenantAPITokenService) List(tenantID string) ([]domain.TenantAPIToken, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	repo, ok := s.repo.(secureTenantAPITokenRepo)
	if !ok {
		return []domain.TenantAPIToken{}, nil
	}
	tokens, err := repo.ListActiveByTenant(tenantID)
	if err != nil {
		return nil, err
	}
	for i := range tokens {
		tokens[i].MaskedToken = tokens[i].TokenPrefix + "••••••••••••"
		tokens[i].EncryptedToken = ""
	}
	return tokens, nil
}

func (s *TenantAPITokenService) Reveal(tenantID, environment string) (*domain.TenantAPIToken, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	environment = strings.ToUpper(strings.TrimSpace(environment))
	if environment != "HML" && environment != "PRODUCTION" {
		return nil, ErrValidation
	}
	repo, ok := s.repo.(secureTenantAPITokenRepo)
	if !ok {
		return nil, ErrValidation
	}
	token, err := repo.FindActiveByTenantEnvironment(tenantID, environment)
	if err != nil {
		return nil, err
	}
	if token.EncryptedToken == "" {
		token.MaskedToken = token.TokenPrefix + "••••••••••••"
		return token, nil
	}
	token.Token, err = s.decrypt(token.EncryptedToken)
	token.EncryptedToken = ""
	return token, err
}

func (s *TenantAPITokenService) encrypt(plain string) (string, error) {
	if len(s.encryptionKey) == 0 {
		return "", nil
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}
func (s *TenantAPITokenService) decrypt(encoded string) (string, error) {
	if len(s.encryptionKey) == 0 {
		return "", ErrValidation
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", ErrValidation
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	return string(plain), err
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
