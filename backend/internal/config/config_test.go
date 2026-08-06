package config

import "testing"

func TestValidateAuthConfig(t *testing.T) {
	valid := &Config{
		Env:                "production",
		DatabaseURL:        "postgres://user:pass@host:5432/db?sslmode=disable",
		Port:               "8080",
		JWTSecret:          "01234567890123456789012345678901",
		JWTIssuer:          "middleware-boletos",
		JWTAudience:        "middleware-boletos-api",
		CORSAllowedOrigins: []string{"https://app.example.com"},
	}

	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{name: "production without DATABASE_URL", cfg: &Config{Env: "production", JWTSecret: valid.JWTSecret, JWTIssuer: valid.JWTIssuer, JWTAudience: valid.JWTAudience, CORSAllowedOrigins: valid.CORSAllowedOrigins}, want: true},
		{name: "production without JWT_SECRET", cfg: &Config{Env: "production", JWTIssuer: valid.JWTIssuer, JWTAudience: valid.JWTAudience, DatabaseURL: valid.DatabaseURL, Port: valid.Port}, want: true},
		{name: "production with short secret", cfg: &Config{Env: "production", JWTSecret: "short", JWTIssuer: valid.JWTIssuer, JWTAudience: valid.JWTAudience, DatabaseURL: valid.DatabaseURL, Port: valid.Port}, want: true},
		{name: "production without JWT_ISSUER", cfg: &Config{Env: "production", JWTSecret: valid.JWTSecret, JWTAudience: valid.JWTAudience, DatabaseURL: valid.DatabaseURL, Port: valid.Port}, want: true},
		{name: "production without JWT_AUDIENCE", cfg: &Config{Env: "production", JWTSecret: valid.JWTSecret, JWTIssuer: valid.JWTIssuer, DatabaseURL: valid.DatabaseURL, Port: valid.Port}, want: true},
		{name: "production without CORS_ALLOWED_ORIGINS", cfg: &Config{Env: "production", JWTSecret: valid.JWTSecret, JWTIssuer: valid.JWTIssuer, JWTAudience: valid.JWTAudience, DatabaseURL: valid.DatabaseURL, Port: valid.Port}, want: true},
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

func TestParseBoolOnlyTrueEnables(t *testing.T) {
	tests := map[string]bool{
		"true":  true,
		"TRUE":  true,
		"false": false,
		"1":     false,
		"yes":   false,
		"":      false,
	}
	for raw, want := range tests {
		if got := parseBool(raw); got != want {
			t.Fatalf("parseBool(%q) = %v, want %v", raw, got, want)
		}
	}
}
