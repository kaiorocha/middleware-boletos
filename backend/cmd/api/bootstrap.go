package main

import (
	"log"
	"strings"

	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
	"github.com/kaiorocha/middleware-boletos/backend/internal/config"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

func bootstrapPlatformAdmin(cfg *config.Config, userSvc *service.UserService) error {
	if cfg == nil || userSvc == nil {
		return nil
	}
	email := service.NormalizeEmail(cfg.BootstrapAdminEmail)
	password := strings.TrimSpace(cfg.BootstrapAdminPassword)
	name := strings.TrimSpace(cfg.BootstrapAdminName)
	if email == "" || password == "" || name == "" {
		return nil
	}
	if len(password) < 8 {
		return service.ErrValidation
	}

	exists, err := userSvc.HasRole(authn.RolePlatformAdmin)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	hash, err := authn.HashPassword(password)
	if err != nil {
		return err
	}
	user := &domain.User{
		Email:        email,
		Name:         name,
		Status:       "ACTIVE",
		Roles:        []string{authn.RolePlatformAdmin},
		PasswordHash: hash,
	}
	if err := userSvc.Create(user); err != nil {
		return err
	}
	log.Printf("bootstrap platform admin created: %s", email)
	return nil
}
