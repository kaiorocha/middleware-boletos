package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
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
	custRepo := repository.NewCustomerRepo(db)
	boletoRepo := repository.NewBoletoRepo(db)

	// services
	tenantSvc := service.NewTenantService(tenantRepo)
	boletoSvc := service.NewBoletoService(boletoRepo, custRepo)

	app := &App{TenantSvc: tenantSvc, BoletoSvc: boletoSvc, CustRepo: custRepo}

	h := app.routes()
	log.Printf("starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
