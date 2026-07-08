package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type TenantRepo struct {
	db *sql.DB
}

func NewTenantRepo(db *sql.DB) *TenantRepo { return &TenantRepo{db: db} }

func (r *TenantRepo) Create(t *domain.Tenant) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	_, err := r.db.Exec(`INSERT INTO tenants (id,name,owner_id,created_at,updated_at) VALUES ($1,$2,$3,now(),now())`, t.ID, t.Name, t.OwnerID)
	return err
}

func (r *TenantRepo) FindByID(id string) (*domain.Tenant, error) {
	row := r.db.QueryRow(`SELECT id,name,owner_id,created_at,updated_at,deleted_at FROM tenants WHERE id = $1 AND deleted_at IS NULL`, id)
	var t domain.Tenant
	var ownerID sql.NullString
	var deleted *time.Time
	if err := row.Scan(&t.ID, &t.Name, &ownerID, &t.CreatedAt, &t.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	if ownerID.Valid { v := ownerID.String; t.OwnerID = &v }
	if deleted != nil { t.DeletedAt = deleted }
	return &t, nil
}

func (r *TenantRepo) List() ([]domain.Tenant, error) {
	rows, err := r.db.Query(`SELECT id,name,owner_id,created_at,updated_at,deleted_at FROM tenants WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		var ownerID sql.NullString
		var deleted *time.Time
		if err := rows.Scan(&t.ID, &t.Name, &ownerID, &t.CreatedAt, &t.UpdatedAt, &deleted); err != nil {
			return nil, err
		}
		if ownerID.Valid { v := ownerID.String; t.OwnerID = &v }
		if deleted != nil { t.DeletedAt = deleted }
		out = append(out, t)
	}
	return out, nil
}

func (r *TenantRepo) Update(t *domain.Tenant) error {
	_, err := r.db.Exec(`UPDATE tenants SET name = $1, owner_id = $2, updated_at = now() WHERE id = $3 AND deleted_at IS NULL`, t.Name, t.OwnerID, t.ID)
	return err
}

func (r *TenantRepo) Delete(id string) error {
	_, err := r.db.Exec(`UPDATE tenants SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL`, time.Now().UTC(), id)
	return err
}
