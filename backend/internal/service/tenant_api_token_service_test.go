package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type tenantAPITokenRepoFake struct {
	token *domain.TenantAPIToken
	hash  string
}

type secureTenantAPITokenRepoFake struct {
	tenantAPITokenRepoFake
	encrypted string
}

func (r *secureTenantAPITokenRepoFake) RotateSecure(token *domain.TenantAPIToken, hash, encrypted string) error {
	copy := *token
	copy.Token = ""
	r.token, r.hash, r.encrypted = &copy, hash, encrypted
	return nil
}
func (r *secureTenantAPITokenRepoFake) ListActiveByTenant(string) ([]domain.TenantAPIToken, error) {
	token := *r.token
	token.EncryptedToken = r.encrypted
	return []domain.TenantAPIToken{token}, nil
}
func (r *secureTenantAPITokenRepoFake) FindActiveByTenantEnvironment(string, string) (*domain.TenantAPIToken, error) {
	token := *r.token
	token.EncryptedToken = r.encrypted
	return &token, nil
}

func (r *tenantAPITokenRepoFake) Rotate(token *domain.TenantAPIToken, hash string) error {
	r.token, r.hash = token, hash
	return nil
}

func TestTenantAPITokenServiceMasksAndRevealsEncryptedToken(t *testing.T) {
	const tenantID = "550e8400-e29b-41d4-a716-446655440000"
	repo := &secureTenantAPITokenRepoFake{}
	svc := NewTenantAPITokenService(repo, "a-stable-secret-used-to-encrypt-api-tokens")
	issued, err := svc.Issue(tenantID, "HML")
	if err != nil {
		t.Fatal(err)
	}
	if repo.encrypted == "" || strings.Contains(repo.encrypted, issued.Token) {
		t.Fatal("token must be stored encrypted")
	}
	listed, err := svc.List(tenantID)
	if err != nil || len(listed) != 1 || listed[0].Token != "" || listed[0].MaskedToken == "" {
		t.Fatalf("unexpected masked tokens: %+v %v", listed, err)
	}
	revealed, err := svc.Reveal(tenantID, "hml")
	if err != nil || revealed.Token != issued.Token {
		t.Fatalf("unexpected revealed token: %+v %v", revealed, err)
	}
}
func (r *tenantAPITokenRepoFake) FindActiveByHash(hash string) (*domain.TenantAPIToken, error) {
	if hash != r.hash {
		return nil, errors.New("not found")
	}
	return r.token, nil
}

func TestTenantAPITokenServiceIssuesAndAuthenticatesEnvironmentTokens(t *testing.T) {
	tenantID := "550e8400-e29b-41d4-a716-446655440000"
	for _, environment := range []string{"HML", "PRODUCTION"} {
		t.Run(environment, func(t *testing.T) {
			repo := &tenantAPITokenRepoFake{}
			svc := NewTenantAPITokenService(repo)
			issued, err := svc.Issue(tenantID, environment)
			if err != nil {
				t.Fatalf("issue token: %v", err)
			}
			prefix := "giga_hml_"
			if environment == "PRODUCTION" {
				prefix = "giga_prod_"
			}
			if !strings.HasPrefix(issued.Token, prefix) || issued.TenantID != tenantID {
				t.Fatalf("unexpected token: %+v", issued)
			}
			if repo.hash == "" || strings.Contains(repo.hash, issued.Token) {
				t.Fatal("repository must receive only a one-way hash")
			}
			authenticated, err := svc.Authenticate(issued.Token)
			if err != nil || authenticated.TenantID != tenantID {
				t.Fatalf("authenticate token: %+v %v", authenticated, err)
			}
		})
	}
}
