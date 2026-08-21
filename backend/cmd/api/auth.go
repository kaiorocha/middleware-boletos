package main

import (
	"net/http"
	"os"
	"strings"

	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

type AuthDecision struct {
	Authenticated bool
	Allowed       bool
}

type TenantAuthorizer interface {
	AuthorizeTenant(*http.Request, string) AuthDecision
}

type IdentityTenantAuthorizer struct{}

func NewIdentityTenantAuthorizer() *IdentityTenantAuthorizer {
	return &IdentityTenantAuthorizer{}
}

func (a *IdentityTenantAuthorizer) AuthorizeTenant(r *http.Request, tenantID string) AuthDecision {
	identity, ok := authn.IdentityFromContext(r.Context())
	if !ok || identity.UserID == "" {
		return AuthDecision{Authenticated: false, Allowed: false}
	}
	return AuthDecision{Authenticated: true, Allowed: identity.HasTenant(tenantID)}
}

type RequestAuthenticator struct {
	env       string
	validator authn.TokenValidator
	apiTokens interface {
		Authenticate(string) (*domain.TenantAPIToken, error)
	}
}

func NewRequestAuthenticator(env string, validator authn.TokenValidator) *RequestAuthenticator {
	env = strings.ToLower(strings.TrimSpace(env))
	if env == "" {
		env = "production"
	}
	return &RequestAuthenticator{env: env, validator: validator}
}

func (a *RequestAuthenticator) WithTenantAPITokens(tokens interface {
	Authenticate(string) (*domain.TenantAPIToken, error)
}) *RequestAuthenticator {
	a.apiTokens = tokens
	return a
}

func (a *RequestAuthenticator) Authenticate(r *http.Request) (authn.Identity, bool) {
	if a != nil && a.env == "development" {
		if identity, ok := identityFromDevelopmentHeaders(r); ok {
			return identity, true
		}
	}

	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return authn.Identity{}, false
	}
	token, ok := bearerToken(authorization)
	if !ok || a == nil {
		return authn.Identity{}, false
	}
	if a.validator != nil {
		if identity, err := a.validator.Validate(r.Context(), token); err == nil {
			return identity, true
		}
	}
	if a.apiTokens != nil {
		apiToken, err := a.apiTokens.Authenticate(token)
		if err == nil && apiToken != nil && tokenEnvironmentAllowed(a.env, apiToken.Environment) {
			return authn.Identity{UserID: apiToken.ID, TenantIDs: []string{apiToken.TenantID}, Roles: []string{authn.RoleTenantAdmin, authn.RoleTenantAPI}}, true
		}
	}
	return authn.Identity{}, false
}

func tokenEnvironmentAllowed(appEnv, tokenEnv string) bool {
	if strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return tokenEnv == "PRODUCTION"
	}
	return tokenEnv == "HML"
}

func identityFromDevelopmentHeaders(r *http.Request) (authn.Identity, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-Dev-User-ID"))
	tenantID := strings.TrimSpace(r.Header.Get("X-Dev-Tenant-ID"))
	if !service.IsValidUUID(userID) || !service.IsValidUUID(tenantID) {
		return authn.Identity{}, false
	}
	return authn.Identity{UserID: userID, TenantIDs: []string{tenantID}}, true
}

func bearerToken(authorization string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
	return token, token != ""
}

func (a *App) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r) {
			next.ServeHTTP(w, r)
			return
		}
		identity, ok := a.requestAuthenticator().Authenticate(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
			return
		}
		if identity.HasRole(authn.RoleTenantAPI) && !isTenantPublicAPIRoute(r.URL.Path) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "route not available for tenant API token")
			return
		}
		next.ServeHTTP(w, r.WithContext(authn.WithIdentity(r.Context(), identity)))
	})
}

func isTenantPublicAPIRoute(path string) bool {
	return path == "/api/v1/boletos" || strings.HasPrefix(path, "/api/v1/boletos/") || path == "/api/v1/transactions" || path == "/api/v1/blocked-emails" || strings.HasPrefix(path, "/api/v1/blocked-emails/")
}

func (a *App) requestAuthenticator() *RequestAuthenticator {
	if a.Authenticator != nil {
		return a.Authenticator
	}
	return NewRequestAuthenticator(defaultAppEnv(), nil)
}

func isPublicRoute(r *http.Request) bool {
	if r.Method == http.MethodGet && (r.URL.Path == "/health" || r.URL.Path == "/ready") {
		return true
	}
	return r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login"
}

func defaultAppEnv() string {
	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	if env != "" {
		return env
	}
	env = strings.TrimSpace(os.Getenv("BACKEND_ENV"))
	if env != "" {
		return env
	}
	return "production"
}
