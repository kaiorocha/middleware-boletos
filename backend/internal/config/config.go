package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds application configuration
type Config struct {
	Port                   string
	DatabaseURL            string
	RedisURL               string
	Env                    string
	JWTSecret              string
	JWTIssuer              string
	JWTAudience            string
	EnableAdminBootstrap   bool
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
	BootstrapAdminName     string
	CORSAllowedOrigins     []string
}

const MinJWTSecretLength = 32

// Load reads configuration from environment with sensible defaults
func Load() *Config {
	cfg := &Config{
		Port:                   getEnv("PORT", "8080"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/middleware?sslmode=disable"),
		RedisURL:               getEnv("REDIS_URL", "redis://redis:6379/0"),
		Env:                    getEnv("APP_ENV", getEnv("BACKEND_ENV", "production")),
		JWTSecret:              getEnv("JWT_SECRET", ""),
		JWTIssuer:              getEnv("JWT_ISSUER", ""),
		JWTAudience:            getEnv("JWT_AUDIENCE", ""),
		EnableAdminBootstrap:   parseBool(getEnv("ENABLE_ADMIN_BOOTSTRAP", "false")),
		BootstrapAdminEmail:    getEnv("BOOTSTRAP_ADMIN_EMAIL", ""),
		BootstrapAdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		BootstrapAdminName:     getEnv("BOOTSTRAP_ADMIN_NAME", ""),
		CORSAllowedOrigins:     splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "")),
	}
	return cfg
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseBool(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func ValidateAuthConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if !strings.EqualFold(strings.TrimSpace(cfg.Env), "production") {
		return nil
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return errors.New("JWT_SECRET is required")
	}
	if len([]byte(cfg.JWTSecret)) < MinJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters", MinJWTSecretLength)
	}
	if strings.TrimSpace(cfg.JWTIssuer) == "" {
		return errors.New("JWT_ISSUER is required")
	}
	if strings.TrimSpace(cfg.JWTAudience) == "" {
		return errors.New("JWT_AUDIENCE is required")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return errors.New("CORS_ALLOWED_ORIGINS is required in production")
	}
	return nil
}
