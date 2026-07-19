package config

import "testing"

func TestValidateAuthConfig(t *testing.T) {
	valid := &Config{
		Env:         "production",
		JWTSecret:   "01234567890123456789012345678901",
		JWTIssuer:   "middleware-boletos",
		JWTAudience: "middleware-boletos-api",
	}

	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{name: "production without JWT_SECRET", cfg: &Config{Env: "production", JWTIssuer: valid.JWTIssuer, JWTAudience: valid.JWTAudience}, want: true},
		{name: "production with short secret", cfg: &Config{Env: "production", JWTSecret: "short", JWTIssuer: valid.JWTIssuer, JWTAudience: valid.JWTAudience}, want: true},
		{name: "production without JWT_ISSUER", cfg: &Config{Env: "production", JWTSecret: valid.JWTSecret, JWTAudience: valid.JWTAudience}, want: true},
		{name: "production without JWT_AUDIENCE", cfg: &Config{Env: "production", JWTSecret: valid.JWTSecret, JWTIssuer: valid.JWTIssuer}, want: true},
		{name: "production complete", cfg: valid, want: false},
		{name: "development without JWT", cfg: &Config{Env: "development"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthConfig(tt.cfg)
			if (err != nil) != tt.want {
				t.Fatalf("expected error=%v, got %v", tt.want, err)
			}
		})
	}
}
