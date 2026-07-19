package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

const hs256 = "HS256"

type TokenValidator interface {
	Validate(context.Context, string) (Identity, error)
}

type JWTConfig struct {
	Secret   string
	Issuer   string
	Audience string
	Now      func() time.Time
}

type HMACValidator struct {
	secret   []byte
	issuer   string
	audience string
	now      func() time.Time
}

func NewHMACValidator(cfg JWTConfig) (*HMACValidator, error) {
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, errors.New("jwt secret is required")
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &HMACValidator{
		secret:   []byte(cfg.Secret),
		issuer:   strings.TrimSpace(cfg.Issuer),
		audience: strings.TrimSpace(cfg.Audience),
		now:      now,
	}, nil
}

func (v *HMACValidator) Validate(_ context.Context, token string) (Identity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Identity{}, ErrInvalidToken
	}

	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if err := decodeJSON(parts[0], &header); err != nil {
		return Identity{}, ErrInvalidToken
	}
	if header.Algorithm != hs256 {
		return Identity{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expected := hmacSHA256(signingInput, v.secret)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(expected, actual) {
		return Identity{}, ErrInvalidToken
	}

	var claims jwtClaims
	if err := decodeJSON(parts[1], &claims); err != nil {
		return Identity{}, ErrInvalidToken
	}
	return v.identityFromClaims(claims)
}

type jwtClaims struct {
	Subject   string          `json:"sub"`
	TenantID  string          `json:"tenant_id"`
	TenantIDs []string        `json:"tenant_ids"`
	Roles     []string        `json:"roles"`
	ExpiresAt int64           `json:"exp"`
	Issuer    string          `json:"iss"`
	Audience  json.RawMessage `json:"aud"`
}

func (v *HMACValidator) identityFromClaims(claims jwtClaims) (Identity, error) {
	if !service.IsValidUUID(claims.Subject) {
		return Identity{}, ErrInvalidToken
	}
	if claims.ExpiresAt <= 0 || !v.now().Before(time.Unix(claims.ExpiresAt, 0)) {
		return Identity{}, ErrInvalidToken
	}
	if v.issuer != "" && claims.Issuer != v.issuer {
		return Identity{}, ErrInvalidToken
	}
	if v.audience != "" && !audienceContains(claims.Audience, v.audience) {
		return Identity{}, ErrInvalidToken
	}

	tenants := make([]string, 0, len(claims.TenantIDs)+1)
	if strings.TrimSpace(claims.TenantID) != "" {
		tenants = append(tenants, strings.TrimSpace(claims.TenantID))
	}
	for _, tenantID := range claims.TenantIDs {
		tenantID = strings.TrimSpace(tenantID)
		if tenantID != "" {
			tenants = append(tenants, tenantID)
		}
	}
	tenants = uniqueValidUUIDs(tenants)
	if len(tenants) == 0 {
		return Identity{}, ErrInvalidToken
	}

	return Identity{UserID: claims.Subject, TenantIDs: tenants, Roles: NormalizeRoles(claims.Roles)}, nil
}

func uniqueValidUUIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !service.IsValidUUID(value) {
			return nil
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func decodeJSON(segment string, out any) error {
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func hmacSHA256(data string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}

func audienceContains(raw json.RawMessage, expected string) bool {
	if len(raw) == 0 {
		return false
	}

	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return single == expected
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return false
	}
	return slicesContains(list, expected)
}

func slicesContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
