package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type WebhookEventRepo struct{ db *sql.DB }

func NewWebhookEventRepo(db *sql.DB) *WebhookEventRepo { return &WebhookEventRepo{db: db} }

func (r *WebhookEventRepo) Create(e *domain.WebhookEvent) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	_, err := r.db.Exec(`INSERT INTO webhook_events (id,tenant_id,type,payload,created_at) VALUES ($1,$2,$3,$4,now())`, e.ID, e.TenantID, e.Type, e.Payload)
	return err
}

func (r *WebhookEventRepo) FindByID(id string) (*domain.WebhookEvent, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,type,payload,created_at FROM webhook_events WHERE id = $1`, id)
	var e domain.WebhookEvent
	if err := row.Scan(&e.ID, &e.TenantID, &e.Type, &e.Payload, &e.CreatedAt); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *WebhookEventRepo) ListByTenant(tenantID string) ([]domain.WebhookEvent, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,type,payload,created_at FROM webhook_events WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.WebhookEvent
	for rows.Next() {
		var e domain.WebhookEvent
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Type, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (r *WebhookEventRepo) Update(e *domain.WebhookEvent) error {
	_, err := r.db.Exec(`UPDATE webhook_events SET type = $1, payload = $2 WHERE id = $3`, e.Type, e.Payload, e.ID)
	return err
}

func (r *WebhookEventRepo) Delete(id string, tenantID string) error {
	_, err := r.db.Exec(`DELETE FROM webhook_events WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *WebhookEventRepo) DeleteBefore(tenantID string, before time.Time) error {
	_, err := r.db.Exec(`DELETE FROM webhook_events WHERE tenant_id = $1 AND created_at < $2`, tenantID, before.UTC())
	return err
}
