package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type AuditLogRepo struct{ db *sql.DB }

func NewAuditLogRepo(db *sql.DB) *AuditLogRepo { return &AuditLogRepo{db: db} }

func (r *AuditLogRepo) Create(a *domain.AuditLog) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	_, err := r.db.Exec(`INSERT INTO audit_logs (id,tenant_id,user_id,action,metadata,created_at) VALUES ($1,$2,$3,$4,$5,now())`, a.ID, a.TenantID, a.UserID, a.Action, a.Metadata)
	return err
}

func (r *AuditLogRepo) FindByID(id string) (*domain.AuditLog, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,user_id,action,metadata,created_at FROM audit_logs WHERE id = $1`, id)
	var a domain.AuditLog
	var userID sql.NullString
	var metadata sql.NullString
	if err := row.Scan(&a.ID, &a.TenantID, &userID, &a.Action, &metadata, &a.CreatedAt); err != nil {
		return nil, err
	}
	if userID.Valid {
		v := userID.String
		a.UserID = &v
	}
	if metadata.Valid {
		v := metadata.String
		a.Metadata = &v
	}
	return &a, nil
}

func (r *AuditLogRepo) ListByTenant(tenantID string) ([]domain.AuditLog, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,user_id,action,metadata,created_at FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AuditLog
	for rows.Next() {
		var a domain.AuditLog
		var userID sql.NullString
		var metadata sql.NullString
		if err := rows.Scan(&a.ID, &a.TenantID, &userID, &a.Action, &metadata, &a.CreatedAt); err != nil {
			return nil, err
		}
		if userID.Valid {
			v := userID.String
			a.UserID = &v
		}
		if metadata.Valid {
			v := metadata.String
			a.Metadata = &v
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *AuditLogRepo) Update(a *domain.AuditLog) error {
	_, err := r.db.Exec(`UPDATE audit_logs SET action = $1, metadata = $2 WHERE id = $3`, a.Action, a.Metadata, a.ID)
	return err
}

func (r *AuditLogRepo) Delete(id string, tenantID string) error {
	_, err := r.db.Exec(`DELETE FROM audit_logs WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *AuditLogRepo) DeleteBefore(tenantID string, before time.Time) error {
	_, err := r.db.Exec(`DELETE FROM audit_logs WHERE tenant_id = $1 AND created_at < $2`, tenantID, before.UTC())
	return err
}
