package auth

import (
	"context"
	"errors"
	"slices"
	"strings"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrNoIdentity   = errors.New("identity not found")
)

const (
	RolePlatformAdmin = "PLATFORM_ADMIN"
	RoleTenantAdmin   = "TENANT_ADMIN"
	RoleTenantUser    = "TENANT_USER"
)

type Identity struct {
	UserID    string
	TenantIDs []string
	Roles     []string
}

func (i Identity) HasTenant(tenantID string) bool {
	return slices.Contains(i.TenantIDs, tenantID)
}

func (i Identity) HasRole(role string) bool {
	return slices.Contains(i.Roles, NormalizeRole(role))
}

func NormalizeRole(role string) string {
	return strings.ToUpper(strings.TrimSpace(role))
}

func NormalizeRoles(roles []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		role = NormalizeRole(role)
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	return out
}

type contextKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(contextKey{}).(Identity)
	if !ok {
		return Identity{}, false
	}
	return identity, true
}
