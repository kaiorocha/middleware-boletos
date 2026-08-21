package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

const (
	testSecret = "01234567890123456789012345678901"
	testIssuer = "middleware-boletos"
	testAud    = "middleware-boletos-api"
	testUser   = "550e8400-e29b-41d4-a716-446655449999"
	testTenant = "550e8400-e29b-41d4-a716-446655440000"
)

func TestHMACValidatorClaimsAndRoles(t *testing.T) {
	validator, err := NewHMACValidator(JWTConfig{Secret: testSecret, Issuer: testIssuer, Audience: testAud})
	if err != nil {
		t.Fatalf("validator: %v", err)
	}

	token := testToken(map[string]any{
		"sub":        testUser,
		"tenant_id":  testTenant,
		"tenant_ids": []string{testTenant, testTenant},
		"roles":      []string{"platform_admin", "PLATFORM_ADMIN", "tenant_admin"},
		"exp":        time.Now().Add(time.Hour).Unix(),
		"iss":        testIssuer,
		"aud":        testAud,
	}, testSecret)

	identity, err := validator.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !identity.HasRole(RolePlatformAdmin) || !identity.HasRole("tenant_admin") {
		t.Fatalf("expected normalized roles, got %+v", identity.Roles)
	}
	if len(identity.Roles) != 2 {
		t.Fatalf("expected duplicate roles removed, got %+v", identity.Roles)
	}
	if !identity.HasTenant(testTenant) || len(identity.TenantIDs) != 1 {
		t.Fatalf("expected duplicate tenants removed, got %+v", identity.TenantIDs)
	}
}

func TestHMACValidatorRejectsInvalidClaims(t *testing.T) {
	validator, err := NewHMACValidator(JWTConfig{Secret: testSecret, Issuer: testIssuer, Audience: testAud})
	if err != nil {
		t.Fatalf("validator: %v", err)
	}

	tests := []struct {
		name   string
		claims map[string]any
	}{
		{
			name: "invalid sub",
			claims: map[string]any{
				"sub": "not-a-uuid", "tenant_id": testTenant, "exp": time.Now().Add(time.Hour).Unix(), "iss": testIssuer, "aud": testAud,
			},
		},
		{
			name: "invalid tenant_id",
			claims: map[string]any{
				"sub": testUser, "tenant_id": "not-a-uuid", "exp": time.Now().Add(time.Hour).Unix(), "iss": testIssuer, "aud": testAud,
			},
		},
		{
			name: "invalid tenant_ids",
			claims: map[string]any{
				"sub": testUser, "tenant_ids": []string{testTenant, "not-a-uuid"}, "exp": time.Now().Add(time.Hour).Unix(), "iss": testIssuer, "aud": testAud,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.Validate(context.Background(), testToken(tt.claims, testSecret))
			if err == nil {
				t.Fatal("expected invalid token")
			}
		})
	}
}

func TestPasswordHashAndCompare(t *testing.T) {
	hash, err := HashPassword("ChangeMe123456!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "ChangeMe123456!" {
		t.Fatal("password hash must not equal plain password")
	}
	if !ComparePassword(hash, "ChangeMe123456!") {
		t.Fatal("expected password to match")
	}
	if ComparePassword(hash, "wrong-password") {
		t.Fatal("expected wrong password to fail")
	}
}

func testToken(claims map[string]any, secret string) string {
	unsigned := jwtPart(map[string]any{"alg": "HS256", "typ": "JWT"}) + "." + jwtPart(claims)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func jwtPart(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}
