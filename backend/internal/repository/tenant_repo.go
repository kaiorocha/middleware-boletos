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
	row := r.db.QueryRow(`SELECT id,name,owner_id,created_at,updated_at,deleted_at FROM tenants WHERE id = $1`, id)
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
	rows, err := r.db.Query(`SELECT id,name,owner_id,created_at,updated_at,deleted_at FROM tenants ORDER BY created_at DESC`)
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
