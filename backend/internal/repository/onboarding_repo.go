package repository

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type OnboardingRepo struct {
	db *sql.DB
}

func NewOnboardingRepo(db *sql.DB) *OnboardingRepo { return &OnboardingRepo{db: db} }

func (r *OnboardingRepo) CreateTenantOnboarding(input domain.OnboardingInput) (*domain.OnboardingResult, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	tenant := input.Tenant
	if tenant.ID == "" {
		tenant.ID = uuid.New().String()
	}
	err = tx.QueryRow(`INSERT INTO tenants (id,name,document,address,district,city,postal_code,state,country_code,area_code,phone_number,webhook_url,owner_id,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,now(),now()) RETURNING created_at,updated_at`, tenant.ID, tenant.Name, tenant.Document, tenant.Address, tenant.District, tenant.City, tenant.PostalCode, tenant.State, tenant.CountryCode, tenant.AreaCode, tenant.PhoneNumber, tenant.WebhookURL, tenant.OwnerID).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)
	if err != nil {
		return nil, err
	}

	var admin *domain.User
	if input.Admin != nil {
		next := *input.Admin
		if next.ID == "" {
			next.ID = uuid.New().String()
		}
		next.TenantID = tenant.ID
		if next.Status == "" {
			next.Status = "ACTIVE"
		}
		next.Roles = authn.NormalizeRoles(next.Roles)
		roles, marshalErr := json.Marshal(next.Roles)
		if marshalErr != nil {
			err = marshalErr
			return nil, err
		}
		err = tx.QueryRow(`INSERT INTO users (id,tenant_id,email,name,status,external_id,password_hash,roles,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,now(),now()) RETURNING created_at,updated_at`, next.ID, next.TenantID, next.Email, next.Name, next.Status, next.ExternalID, next.PasswordHash, string(roles)).Scan(&next.CreatedAt, &next.UpdatedAt)
		if err != nil {
			err = translatePostgresError(err)
			return nil, err
		}
		admin = &next
		tenant.OwnerID = &next.ID
		if _, err = tx.Exec(`UPDATE tenants SET owner_id = $1, updated_at = now() WHERE id = $2`, next.ID, tenant.ID); err != nil {
			return nil, err
		}
	}

	assignments := make([]domain.TenantProvider, 0, len(input.Providers))
	for _, provider := range input.Providers {
		var active bool
		if err = tx.QueryRow(`SELECT status = 'ACTIVE' FROM providers WHERE id = $1 AND tenant_id IS NULL AND deleted_at IS NULL`, provider.ProviderID).Scan(&active); err != nil {
			return nil, err
		}
		if !active {
			err = sql.ErrNoRows
			return nil, err
		}
		assignment := domain.TenantProvider{
			ID:         uuid.New().String(),
			TenantID:   tenant.ID,
			ProviderID: provider.ProviderID,
			Active:     provider.Active,
			Config:     provider.Config,
		}
		if !assignment.Active {
			assignment.Active = true
		}
		err = tx.QueryRow(`
			INSERT INTO tenant_providers (id, tenant_id, provider_id, active, config, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,now(),now())
			RETURNING created_at, updated_at
		`, assignment.ID, assignment.TenantID, assignment.ProviderID, assignment.Active, assignment.Config).Scan(&assignment.CreatedAt, &assignment.UpdatedAt)
		if err != nil {
			err = translatePostgresError(err)
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	err = nil
	return &domain.OnboardingResult{Tenant: tenant, Admin: admin, Providers: assignments}, nil
}
