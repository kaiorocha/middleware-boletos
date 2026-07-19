package main

import (
	"log"
	"net/http"
	"strings"

	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/factory"
	"github.com/kaiorocha/middleware-boletos/backend/internal/repository"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
	"github.com/kaiorocha/middleware-boletos/backend/internal/storage"
)

func main() {
	cfg := config.Load()
	if err := config.ValidateAuthConfig(cfg); err != nil {
		log.Fatalf("auth config invalid: %v", err)
	}
	var jwtValidator authn.TokenValidator
	var jwtIssuer authn.TokenIssuer
	if strings.TrimSpace(cfg.JWTSecret) != "" {
		validator, err := authn.NewHMACValidator(authn.JWTConfig{
			Secret:   cfg.JWTSecret,
			Issuer:   cfg.JWTIssuer,
			Audience: cfg.JWTAudience,
		})
		if err != nil {
			log.Fatalf("auth config invalid: %v", err)
		}
		jwtValidator = validator
		jwtIssuer = validator
	}
	db, err := storage.Connect(cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	// repos
	tenantRepo := repository.NewTenantRepo(db)
	userRepo := repository.NewUserRepo(db)
	custRepo := repository.NewCustomerRepo(db)
	providerRepo := repository.NewProviderRepo(db)
	boletoRepo := repository.NewBoletoRepo(db)
	blacklistRepo := repository.NewBlacklistRepo(db)
	auditRepo := repository.NewAuditLogRepo(db)

	// services
	tenantSvc := service.NewTenantService(tenantRepo)
	userSvc := service.NewUserService(userRepo)
	customerSvc := service.NewCustomerService(custRepo)
	providerSvc := service.NewProviderService(providerRepo)
	blacklistSvc := service.NewBlacklistService(blacklistRepo).WithAuditRepository(auditRepo)
	providerFactory := factory.NewProviderFactory()
	boletoSvc := service.NewBoletoService(boletoRepo).
		WithCustomerRepository(custRepo).
		WithProviderRepository(providerRepo).
		WithBlacklistService(blacklistSvc).
		WithProviderFactory(providerFactory)

	if err := bootstrapPlatformAdmin(cfg, userSvc); err != nil {
		log.Fatalf("bootstrap platform admin: %v", err)
	}

	app := &App{
		TenantSvc:     tenantSvc,
		UserSvc:       userSvc,
		CustomerSvc:   customerSvc,
		ProviderSvc:   providerSvc,
		BoletoSvc:     boletoSvc,
		BlacklistSvc:  blacklistSvc,
		Factory:       providerFactory,
		Authorizer:    NewIdentityTenantAuthorizer(),
		Authenticator: NewRequestAuthenticator(cfg.Env, jwtValidator),
		TokenIssuer:   jwtIssuer,
	}

	h := app.routes()
	log.Printf("starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
