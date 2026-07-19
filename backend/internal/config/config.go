package config

import (
	"os"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	Env         string
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string
}

// Load reads configuration from environment with sensible defaults
func Load() *Config {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/middleware?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://redis:6379/0"),
		Env:         getEnv("APP_ENV", getEnv("BACKEND_ENV", "production")),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		JWTIssuer:   getEnv("JWT_ISSUER", ""),
		JWTAudience: getEnv("JWT_AUDIENCE", ""),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
