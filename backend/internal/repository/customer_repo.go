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
	if c.ID == "" { c.ID = uuid.New().String() }
	_, err := r.db.Exec(`INSERT INTO customers (id,tenant_id,name,document,created_at,updated_at) VALUES ($1,$2,$3,$4,now(),now())`, c.ID, c.TenantID, c.Name, c.Document)
	return err
}

func (r *CustomerRepo) FindByID(id string) (*domain.Customer, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,name,document,created_at,updated_at,deleted_at FROM customers WHERE id = $1`, id)
	var c domain.Customer
	var document sql.NullString
	var deleted *time.Time
	if err := row.Scan(&c.ID,&c.TenantID,&c.Name,&document,&c.CreatedAt,&c.UpdatedAt,&deleted); err != nil { return nil, err }
	if document.Valid { v := document.String; c.Document = &v }
	if deleted != nil { c.DeletedAt = deleted }
	return &c, nil
}

func (r *CustomerRepo) ListByTenant(tenantID string) ([]domain.Customer, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,name,document,created_at,updated_at,deleted_at FROM customers WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []domain.Customer
	for rows.Next() {
		var c domain.Customer
		var document sql.NullString
		var deleted *time.Time
		if err := rows.Scan(&c.ID,&c.TenantID,&c.Name,&document,&c.CreatedAt,&c.UpdatedAt,&deleted); err != nil { return nil, err }
		if document.Valid { v := document.String; c.Document = &v }
		if deleted != nil { c.DeletedAt = deleted }
		out = append(out, c)
	}
	return out, nil
}
