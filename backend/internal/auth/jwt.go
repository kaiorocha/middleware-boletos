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

type TokenIssuer interface {
	Sign(TokenClaims) (string, error)
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

type TokenClaims struct {
	UserID    string
	TenantIDs []string
	Roles     []string
	ExpiresAt time.Time
	Issuer    string
	Audience  string
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

func (v *HMACValidator) Sign(claims TokenClaims) (string, error) {
	tenantIDs := uniqueValidUUIDs(claims.TenantIDs)
	if !service.IsValidUUID(claims.UserID) || len(tenantIDs) != len(claims.TenantIDs) {
		return "", ErrInvalidToken
	}
	if claims.ExpiresAt.IsZero() {
		claims.ExpiresAt = v.now().Add(time.Hour)
	}
	issuer := strings.TrimSpace(claims.Issuer)
	if issuer == "" {
		issuer = v.issuer
	}
	audience := strings.TrimSpace(claims.Audience)
	if audience == "" {
		audience = v.audience
	}

	payload := map[string]any{
		"sub":        claims.UserID,
		"tenant_ids": tenantIDs,
		"roles":      NormalizeRoles(claims.Roles),
		"exp":        claims.ExpiresAt.Unix(),
	}
	if len(tenantIDs) == 1 {
		payload["tenant_id"] = tenantIDs[0]
	}
	if issuer != "" {
		payload["iss"] = issuer
	}
	if audience != "" {
		payload["aud"] = audience
	}

	header, err := encodeJSON(map[string]string{"alg": hs256, "typ": "JWT"})
	if err != nil {
		return "", err
	}
	body, err := encodeJSON(payload)
	if err != nil {
		return "", err
	}
	unsigned := header + "." + body
	signature := base64.RawURLEncoding.EncodeToString(hmacSHA256(unsigned, v.secret))
	return unsigned + "." + signature, nil
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
	roles := NormalizeRoles(claims.Roles)
	if len(tenants) == 0 && !slicesContains(roles, RolePlatformAdmin) {
		return Identity{}, ErrInvalidToken
	}

	return Identity{UserID: claims.Subject, TenantIDs: tenants, Roles: roles}, nil
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

func encodeJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
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
