package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type ProviderRepo struct{ db *sql.DB }

func NewProviderRepo(db *sql.DB) *ProviderRepo { return &ProviderRepo{db: db} }

func (r *ProviderRepo) Create(p *domain.Provider) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.Status == "" {
		p.Status = "ACTIVE"
	}
	var tenantID any
	if p.TenantID != "" {
		tenantID = p.TenantID
	}
	_, err := r.db.Exec(`INSERT INTO providers (id,tenant_id,name,type,status,external_id,config,metadata,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now(),now())`, p.ID, tenantID, p.Name, p.Type, p.Status, p.ExternalID, p.Config, p.Metadata)
	return translatePostgresError(err)
}

func (r *ProviderRepo) FindByID(id string) (*domain.Provider, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,name,type,status,external_id,config,metadata,created_at,updated_at,deleted_at FROM providers WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanProvider(row)
}

func (r *ProviderRepo) ListCatalog() ([]domain.Provider, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,name,type,status,external_id,NULL::text AS config,metadata,created_at,updated_at,deleted_at FROM providers WHERE tenant_id IS NULL AND deleted_at IS NULL ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProviders(rows)
}

func (r *ProviderRepo) ListByTenant(tenantID string) ([]domain.Provider, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT ON (p.id)
			p.id,
			COALESCE(tp.tenant_id, p.tenant_id) AS tenant_id,
			p.name,
			p.type,
			CASE WHEN tp.id IS NOT NULL AND NOT tp.active THEN 'INACTIVE' ELSE p.status END AS status,
			p.external_id,
			NULL::text AS config,
			p.metadata,
			COALESCE(tp.created_at, p.created_at) AS created_at,
			COALESCE(tp.updated_at, p.updated_at) AS updated_at,
			p.deleted_at
		FROM providers p
		LEFT JOIN tenant_providers tp ON tp.provider_id = p.id AND tp.tenant_id = $1 AND tp.deleted_at IS NULL
		WHERE p.deleted_at IS NULL
		  AND (p.tenant_id = $1 OR tp.id IS NOT NULL)
		ORDER BY p.id, p.name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProviders(rows)
}

func (r *ProviderRepo) Update(p *domain.Provider) error {
	_, err := r.db.Exec(`UPDATE providers SET name = $1, type = $2, status = $3, external_id = $4, config = $5, metadata = $6, updated_at = now() WHERE id = $7 AND deleted_at IS NULL`, p.Name, p.Type, p.Status, p.ExternalID, p.Config, p.Metadata, p.ID)
	return translatePostgresError(err)
}

func (r *ProviderRepo) Delete(id string, tenantID string) error {
	if tenantID != "" {
		_, err := r.db.Exec(`UPDATE tenant_providers SET deleted_at = $1, updated_at = $1, active = false WHERE provider_id = $2 AND tenant_id = $3 AND deleted_at IS NULL`, time.Now().UTC(), id, tenantID)
		return err
	}
	_, err := r.db.Exec(`UPDATE providers SET deleted_at = $1, updated_at = $1, status = 'INACTIVE' WHERE id = $2 AND deleted_at IS NULL`, time.Now().UTC(), id)
	return err
}

func (r *ProviderRepo) AssignToTenant(tenantID, providerID string, active bool, config *string) (*domain.TenantProvider, error) {
	assignment := &domain.TenantProvider{ID: uuid.New().String(), TenantID: tenantID, ProviderID: providerID, Active: active, Config: config}
	_, err := r.db.Exec(`
		INSERT INTO tenant_providers (id, tenant_id, provider_id, active, config, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,now(),now())
		ON CONFLICT (tenant_id, provider_id)
		DO UPDATE SET active = EXCLUDED.active, config = EXCLUDED.config, deleted_at = NULL, updated_at = now()
	`, assignment.ID, tenantID, providerID, active, config)
	if err != nil {
		return nil, translatePostgresError(err)
	}
	return assignment, nil
}

func (r *ProviderRepo) IsAllowedForTenant(tenantID, providerID string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM providers p
			LEFT JOIN tenant_providers tp ON tp.provider_id = p.id AND tp.tenant_id = $1 AND tp.deleted_at IS NULL
			WHERE p.id = $2
			  AND p.deleted_at IS NULL
			  AND p.status = 'ACTIVE'
			  AND (
				p.tenant_id = $1
				OR (tp.id IS NOT NULL AND tp.active = true)
			  )
		)
	`, tenantID, providerID).Scan(&allowed)
	return allowed, err
}

func scanProvider(scanner interface{ Scan(dest ...any) error }) (*domain.Provider, error) {
	var p domain.Provider
	var tenantID sql.NullString
	var providerType sql.NullString
	var externalID sql.NullString
	var config sql.NullString
	var metadata sql.NullString
	var deleted *time.Time
	if err := scanner.Scan(&p.ID, &tenantID, &p.Name, &providerType, &p.Status, &externalID, &config, &metadata, &p.CreatedAt, &p.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	if tenantID.Valid {
		p.TenantID = tenantID.String
	}
	if providerType.Valid {
		p.Type = providerType.String
	}
	if externalID.Valid {
		v := externalID.String
		p.ExternalID = &v
	}
	if config.Valid {
		v := config.String
		p.Config = &v
	}
	if metadata.Valid {
		v := metadata.String
		p.Metadata = &v
	}
	if deleted != nil {
		p.DeletedAt = deleted
	}
	return &p, nil
}

func scanProviders(rows *sql.Rows) ([]domain.Provider, error) {
	var out []domain.Provider
	for rows.Next() {
		provider, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *provider)
	}
	return out, nil
}
