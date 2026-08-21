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

func (r *tenantAPITokenRepoFake) Rotate(token *domain.TenantAPIToken, hash string) error {
	r.token, r.hash = token, hash
	return nil
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
