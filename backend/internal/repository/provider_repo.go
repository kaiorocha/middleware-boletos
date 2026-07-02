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
	_, err := r.db.Exec(`INSERT INTO providers (id,tenant_id,name,status,external_id,config,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,now(),now())`, p.ID, p.TenantID, p.Name, p.Status, p.ExternalID, p.Config)
	return err
}

func (r *ProviderRepo) FindByID(id string) (*domain.Provider, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,name,status,external_id,config,created_at,updated_at,deleted_at FROM providers WHERE id = $1 AND deleted_at IS NULL`, id)
	var p domain.Provider
	var externalID sql.NullString
	var config sql.NullString
	var deleted *time.Time
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Status, &externalID, &config, &p.CreatedAt, &p.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	if externalID.Valid {
		v := externalID.String
		p.ExternalID = &v
	}
	if config.Valid {
		v := config.String
		p.Config = &v
	}
	if deleted != nil {
		p.DeletedAt = deleted
	}
	return &p, nil
}

func (r *ProviderRepo) ListByTenant(tenantID string) ([]domain.Provider, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,name,status,external_id,config,created_at,updated_at,deleted_at FROM providers WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Provider
	for rows.Next() {
		var p domain.Provider
		var externalID sql.NullString
		var config sql.NullString
		var deleted *time.Time
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Status, &externalID, &config, &p.CreatedAt, &p.UpdatedAt, &deleted); err != nil {
			return nil, err
		}
		if externalID.Valid {
			v := externalID.String
			p.ExternalID = &v
		}
		if config.Valid {
			v := config.String
			p.Config = &v
		}
		if deleted != nil {
			p.DeletedAt = deleted
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *ProviderRepo) Update(p *domain.Provider) error {
	_, err := r.db.Exec(`UPDATE providers SET name = $1, status = $2, external_id = $3, config = $4, updated_at = now() WHERE id = $5 AND tenant_id = $6 AND deleted_at IS NULL`, p.Name, p.Status, p.ExternalID, p.Config, p.ID, p.TenantID)
	return err
}

func (r *ProviderRepo) Delete(id string, tenantID string) error {
	_, err := r.db.Exec(`UPDATE providers SET deleted_at = $1, updated_at = $1, status = 'INACTIVE' WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`, time.Now().UTC(), id, tenantID)
	return err
}
