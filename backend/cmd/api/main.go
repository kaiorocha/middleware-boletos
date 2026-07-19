package main

import (
	"log"
	"net/http"

	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/factory"
	"github.com/kaiorocha/middleware-boletos/backend/internal/repository"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
	"github.com/kaiorocha/middleware-boletos/backend/internal/storage"
)

func main() {
	cfg := config.Load()
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

	app := &App{
		TenantSvc:    tenantSvc,
		UserSvc:      userSvc,
		CustomerSvc:  customerSvc,
		ProviderSvc:  providerSvc,
		BoletoSvc:    boletoSvc,
		BlacklistSvc: blacklistSvc,
		Factory:      providerFactory,
	}

	h := app.routes()
	log.Printf("starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
