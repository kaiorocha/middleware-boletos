package auth

import (
	"context"
	"errors"
	"slices"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrNoIdentity   = errors.New("identity not found")
)

type Identity struct {
	UserID    string
	TenantIDs []string
}

func (i Identity) HasTenant(tenantID string) bool {
	return slices.Contains(i.TenantIDs, tenantID)
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
