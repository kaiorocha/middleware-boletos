package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type CustomerRepo struct{ db *sql.DB }

func NewCustomerRepo(db *sql.DB) *CustomerRepo { return &CustomerRepo{db: db} }

func (r *CustomerRepo) Create(c *domain.Customer) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.Status == "" {
		c.Status = "ACTIVE"
	}
	_, err := r.db.Exec(`INSERT INTO customers (id,tenant_id,name,document,status,external_id,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,now(),now())`, c.ID, c.TenantID, c.Name, c.Document, c.Status, c.ExternalID)
	return err
}

func (r *CustomerRepo) FindByID(id string) (*domain.Customer, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,name,document,status,external_id,created_at,updated_at,deleted_at FROM customers WHERE id = $1 AND deleted_at IS NULL`, id)
	var c domain.Customer
	var document sql.NullString
	var externalID sql.NullString
	var deleted *time.Time
	if err := row.Scan(&c.ID, &c.TenantID, &c.Name, &document, &c.Status, &externalID, &c.CreatedAt, &c.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	if document.Valid {
		v := document.String
		c.Document = &v
	}
	if externalID.Valid {
		v := externalID.String
		c.ExternalID = &v
	}
	if deleted != nil {
		c.DeletedAt = deleted
	}
	return &c, nil
}

func (r *CustomerRepo) ListByTenant(tenantID string) ([]domain.Customer, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,name,document,status,external_id,created_at,updated_at,deleted_at FROM customers WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Customer
	for rows.Next() {
		var c domain.Customer
		var document sql.NullString
		var externalID sql.NullString
		var deleted *time.Time
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &document, &c.Status, &externalID, &c.CreatedAt, &c.UpdatedAt, &deleted); err != nil {
			return nil, err
		}
		if document.Valid {
			v := document.String
			c.Document = &v
		}
		if externalID.Valid {
			v := externalID.String
			c.ExternalID = &v
		}
		if deleted != nil {
			c.DeletedAt = deleted
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *CustomerRepo) Update(c *domain.Customer) error {
	_, err := r.db.Exec(`UPDATE customers SET name = $1, document = $2, status = $3, external_id = $4, updated_at = now() WHERE id = $5 AND tenant_id = $6 AND deleted_at IS NULL`, c.Name, c.Document, c.Status, c.ExternalID, c.ID, c.TenantID)
	return err
}

func (r *CustomerRepo) Delete(id string, tenantID string) error {
	_, err := r.db.Exec(`UPDATE customers SET deleted_at = $1, updated_at = $1, status = 'INACTIVE' WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`, time.Now().UTC(), id, tenantID)
	return err
}
