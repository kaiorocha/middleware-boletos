package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/factory"
	"github.com/kaiorocha/middleware-boletos/backend/internal/repository"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
	"github.com/kaiorocha/middleware-boletos/backend/internal/storage"
)

func main() {
	cfg := config.Load()
	logger := slog.Default()
	// starting - do not include secrets
	logger.Info("application_starting", "env", cfg.Env, "addr", ":"+cfg.Port)

	// minimal command parsing: `migrate`
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			// migrations require DB only
			if strings.TrimSpace(cfg.DatabaseURL) == "" {
				logger.Error("migrate_failed", "error", "DATABASE_URL is required")
				os.Exit(1)
			}
			db, err := storage.Connect(cfg)
			if err != nil {
				logger.Error("db_connect_error", "error", err)
				os.Exit(1)
			}
			defer db.Close()
			if err := storage.RunMigrations(db); err != nil {
				logger.Error("migrate_failed", "error", err)
				os.Exit(1)
			}
			logger.Info("migrate_completed")
			os.Exit(0)
		default:
			logger.Error("unknown_command", "cmd", os.Args[1])
			os.Exit(2)
		}
	}

	// normal startup - full validation
	if err := config.ValidateAuthConfig(cfg); err != nil {
		logger.Error("auth_config_invalid", "error", err)
		os.Exit(1)
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
			logger.Error("auth_config_invalid", "error", err)
			os.Exit(1)
		}
		jwtValidator = validator
		jwtIssuer = validator
	}

	db, err := storage.Connect(cfg)
	if err != nil {
		logger.Error("db_connect_error", "error", err)
		os.Exit(1)
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
	onboardingRepo := repository.NewOnboardingRepo(db)

	// services
	tenantSvc := service.NewTenantService(tenantRepo)
	userSvc := service.NewUserService(userRepo)
	customerSvc := service.NewCustomerService(custRepo)
	providerSvc := service.NewProviderService(providerRepo)
	onboardingSvc := service.NewOnboardingService(onboardingRepo)
	blacklistSvc := service.NewBlacklistService(blacklistRepo).WithAuditRepository(auditRepo)
	providerFactory := factory.NewProviderFactory()
	boletoSvc := service.NewBoletoService(boletoRepo).
		WithTenantRepository(tenantRepo).
		WithCustomerRepository(custRepo).
		WithProviderRepository(providerRepo).
		WithBlacklistService(blacklistSvc).
		WithProviderFactory(providerFactory)

	if err := bootstrapPlatformAdmin(cfg, userSvc); err != nil {
		logger.Error("bootstrap_platform_admin_failed", "error", err)
		os.Exit(1)
	}

	app := &App{
		DB:            db,
		TenantSvc:     tenantSvc,
		UserSvc:       userSvc,
		CustomerSvc:   customerSvc,
		ProviderSvc:   providerSvc,
		BoletoSvc:     boletoSvc,
		BlacklistSvc:  blacklistSvc,
		OnboardingSvc: onboardingSvc,
		Factory:       providerFactory,
		Authorizer:    NewIdentityTenantAuthorizer(),
		Authenticator: NewRequestAuthenticator(cfg.Env, jwtValidator),
		TokenIssuer:   jwtIssuer,
		CORSOrigins:   cfg.CORSAllowedOrigins,
	}

	h := app.routes()

	// configured HTTP server with timeouts
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// start server in background
	done := make(chan struct{})
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_failed", "error", err)
			os.Exit(1)
		}
		close(done)
	}()

	// application started - best-effort approximation after ListenAndServe goroutine launched
	logger.Info("application_started", "env", cfg.Env, "addr", srv.Addr)

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("application_shutdown_started", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server_shutdown_error", "error", err)
	}
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("db_close_error", "error", err)
		}
	}
	logger.Info("application_shutdown_completed")
	<-done
}
