package main

import (
	"errors"
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
	production := strings.EqualFold(strings.TrimSpace(cfg.Env), "production")
	if production && !cfg.EnableAdminBootstrap {
		log.Print("platform admin bootstrap disabled")
		return nil
	}
	email := service.NormalizeEmail(cfg.BootstrapAdminEmail)
	password := strings.TrimSpace(cfg.BootstrapAdminPassword)
	name := strings.TrimSpace(cfg.BootstrapAdminName)
	if email == "" || password == "" || name == "" {
		if production && cfg.EnableAdminBootstrap {
			return errors.New("bootstrap admin credentials are required")
		}
		log.Print("platform admin bootstrap disabled")
		return nil
	}
	minPasswordLength := 8
	if production {
		minPasswordLength = 12
	}
	if len(password) < minPasswordLength {
		return service.ErrValidation
	}

	exists, err := userSvc.HasRole(authn.RolePlatformAdmin)
	if err != nil {
		return err
	}
	if exists {
		log.Print("platform admin already exists")
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
	log.Print("platform admin bootstrap created")
	return nil
}
