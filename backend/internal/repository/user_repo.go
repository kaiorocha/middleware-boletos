package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type UserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(u *domain.User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.Status == "" {
		u.Status = "ACTIVE"
	}
	_, err := r.db.Exec(`INSERT INTO users (id,tenant_id,email,name,status,external_id,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,now(),now())`, u.ID, u.TenantID, u.Email, u.Name, u.Status, u.ExternalID)
	return err
}

func (r *UserRepo) FindByID(id string) (*domain.User, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,email,name,status,external_id,created_at,updated_at,deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	var u domain.User
	var externalID sql.NullString
	var deleted *time.Time
	if err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Status, &externalID, &u.CreatedAt, &u.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	if externalID.Valid {
		v := externalID.String
		u.ExternalID = &v
	}
	if deleted != nil {
		u.DeletedAt = deleted
	}
	return &u, nil
}

func (r *UserRepo) ListByTenant(tenantID string) ([]domain.User, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,email,name,status,external_id,created_at,updated_at,deleted_at FROM users WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var u domain.User
		var externalID sql.NullString
		var deleted *time.Time
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Status, &externalID, &u.CreatedAt, &u.UpdatedAt, &deleted); err != nil {
			return nil, err
		}
		if externalID.Valid {
			v := externalID.String
			u.ExternalID = &v
		}
		if deleted != nil {
			u.DeletedAt = deleted
		}
		out = append(out, u)
	}
	return out, nil
}

func (r *UserRepo) Update(u *domain.User) error {
	_, err := r.db.Exec(`UPDATE users SET email = $1, name = $2, status = $3, external_id = $4, updated_at = now() WHERE id = $5 AND tenant_id = $6 AND deleted_at IS NULL`, u.Email, u.Name, u.Status, u.ExternalID, u.ID, u.TenantID)
	return err
}

func (r *UserRepo) Delete(id string, tenantID string) error {
	_, err := r.db.Exec(`UPDATE users SET deleted_at = $1, updated_at = $1, status = 'INACTIVE' WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`, time.Now().UTC(), id, tenantID)
	return err
}
