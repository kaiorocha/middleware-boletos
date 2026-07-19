package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

type AuthDecision struct {
	Authenticated bool
	Allowed       bool
}

type TenantAuthorizer interface {
	AuthorizeTenant(*http.Request, string) AuthDecision
}

type HeaderTenantAuthorizer struct {
	env string
}

func NewHeaderTenantAuthorizer(env string) *HeaderTenantAuthorizer {
	env = strings.ToLower(strings.TrimSpace(env))
	if env == "" {
		env = "production"
	}
	return &HeaderTenantAuthorizer{env: env}
}

func (a *HeaderTenantAuthorizer) AuthorizeTenant(r *http.Request, tenantID string) AuthDecision {
	if a.env == "development" {
		devTenantID := strings.TrimSpace(r.Header.Get("X-Dev-Tenant-ID"))
		if devTenantID != "" {
			return AuthDecision{Authenticated: true, Allowed: devTenantID == tenantID}
		}
	}

	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if !service.IsValidUUID(userID) {
		return AuthDecision{Authenticated: false, Allowed: false}
	}

	allowedTenants := strings.TrimSpace(r.Header.Get("X-Tenant-IDs"))
	if allowedTenants == "" {
		allowedTenants = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	}
	if allowedTenants == "" {
		return AuthDecision{Authenticated: true, Allowed: false}
	}

	for _, candidate := range strings.Split(allowedTenants, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == tenantID && service.IsValidUUID(candidate) {
			return AuthDecision{Authenticated: true, Allowed: true}
		}
	}
	return AuthDecision{Authenticated: true, Allowed: false}
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
