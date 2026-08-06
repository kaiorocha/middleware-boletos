package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
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
	roles, err := json.Marshal(authn.NormalizeRoles(u.Roles))
	if err != nil {
		return err
	}
	var tenantID any
	if u.TenantID != "" {
		tenantID = u.TenantID
	}
	_, err = r.db.Exec(`INSERT INTO users (id,tenant_id,email,name,status,external_id,password_hash,roles,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,now(),now())`, u.ID, tenantID, u.Email, u.Name, u.Status, u.ExternalID, u.PasswordHash, string(roles))
	return translatePostgresError(err)
}

func (r *UserRepo) FindByID(id string) (*domain.User, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,email,name,status,external_id,password_hash,roles,created_at,updated_at,deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanUser(row)
}

func (r *UserRepo) FindByEmail(email string) (*domain.User, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,email,name,status,external_id,password_hash,roles,created_at,updated_at,deleted_at FROM users WHERE lower(email) = lower($1) AND deleted_at IS NULL ORDER BY tenant_id NULLS FIRST, created_at ASC LIMIT 1`, email)
	return scanUser(row)
}

func (r *UserRepo) HasRole(role string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM users
		WHERE deleted_at IS NULL
		  AND roles ? $1
	)`, authn.NormalizeRole(role)).Scan(&exists)
	return exists, err
}

func scanUser(scanner interface {
	Scan(dest ...any) error
}) (*domain.User, error) {
	var u domain.User
	var tenantID sql.NullString
	var externalID sql.NullString
	var passwordHash sql.NullString
	var rolesRaw []byte
	var deleted *time.Time
	if err := scanner.Scan(&u.ID, &tenantID, &u.Email, &u.Name, &u.Status, &externalID, &passwordHash, &rolesRaw, &u.CreatedAt, &u.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	if tenantID.Valid {
		u.TenantID = tenantID.String
	}
	if externalID.Valid {
		v := externalID.String
		u.ExternalID = &v
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if len(rolesRaw) > 0 {
		_ = json.Unmarshal(rolesRaw, &u.Roles)
		u.Roles = authn.NormalizeRoles(u.Roles)
	}
	if deleted != nil {
		u.DeletedAt = deleted
	}
	return &u, nil
}

func (r *UserRepo) ListByTenant(tenantID string) ([]domain.User, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,email,name,status,external_id,password_hash,roles,created_at,updated_at,deleted_at FROM users WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, nil
}

func (r *UserRepo) Update(u *domain.User) error {
	_, err := r.db.Exec(`UPDATE users SET email = $1, name = $2, status = $3, external_id = $4, updated_at = now() WHERE id = $5 AND tenant_id = $6 AND deleted_at IS NULL`, u.Email, u.Name, u.Status, u.ExternalID, u.ID, u.TenantID)
	return translatePostgresError(err)
}

func (r *UserRepo) Delete(id string, tenantID string) error {
	_, err := r.db.Exec(`UPDATE users SET deleted_at = $1, updated_at = $1, status = 'INACTIVE' WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`, time.Now().UTC(), id, tenantID)
	return err
}
