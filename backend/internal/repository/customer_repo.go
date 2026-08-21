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
	_, err := r.db.Exec(`INSERT INTO customers (id,tenant_id,name,document,email,address,number,complement,district,city,state,postal_code,status,external_id,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now(),now())`, c.ID, c.TenantID, c.Name, c.Document, c.Email, c.Address, c.Number, c.Complement, c.District, c.City, c.State, c.PostalCode, c.Status, c.ExternalID)
	return translatePostgresError(err)
}

func (r *CustomerRepo) FindByID(id string) (*domain.Customer, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,name,document,email,address,number,complement,district,city,state,postal_code,status,external_id,created_at,updated_at,deleted_at FROM customers WHERE id = $1 AND deleted_at IS NULL`, id)
	var c domain.Customer
	var document, email, address, number, complement, district, city, state, postalCode, externalID sql.NullString
	var deleted *time.Time
	if err := row.Scan(&c.ID, &c.TenantID, &c.Name, &document, &email, &address, &number, &complement, &district, &city, &state, &postalCode, &c.Status, &externalID, &c.CreatedAt, &c.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	assignCustomerNullableStrings(&c, document, email, address, number, complement, district, city, state, postalCode, externalID)
	if deleted != nil {
		c.DeletedAt = deleted
	}
	return &c, nil
}

func (r *CustomerRepo) ListByTenant(tenantID string) ([]domain.Customer, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,name,document,email,address,number,complement,district,city,state,postal_code,status,external_id,created_at,updated_at,deleted_at FROM customers WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Customer
	for rows.Next() {
		var c domain.Customer
		var document, email, address, number, complement, district, city, state, postalCode, externalID sql.NullString
		var deleted *time.Time
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &document, &email, &address, &number, &complement, &district, &city, &state, &postalCode, &c.Status, &externalID, &c.CreatedAt, &c.UpdatedAt, &deleted); err != nil {
			return nil, err
		}
		assignCustomerNullableStrings(&c, document, email, address, number, complement, district, city, state, postalCode, externalID)
		if deleted != nil {
			c.DeletedAt = deleted
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *CustomerRepo) Update(c *domain.Customer) error {
	_, err := r.db.Exec(`UPDATE customers SET name = $1, document = $2, email = $3, address = $4, number = $5, complement = $6, district = $7, city = $8, state = $9, postal_code = $10, status = $11, external_id = $12, updated_at = now() WHERE id = $13 AND tenant_id = $14 AND deleted_at IS NULL`, c.Name, c.Document, c.Email, c.Address, c.Number, c.Complement, c.District, c.City, c.State, c.PostalCode, c.Status, c.ExternalID, c.ID, c.TenantID)
	return translatePostgresError(err)
}

func (r *CustomerRepo) Delete(id string, tenantID string) error {
	_, err := r.db.Exec(`UPDATE customers SET deleted_at = $1, updated_at = $1, status = 'INACTIVE' WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`, time.Now().UTC(), id, tenantID)
	return err
}

func assignCustomerNullableStrings(c *domain.Customer, document, email, address, number, complement, district, city, state, postalCode, externalID sql.NullString) {
	if document.Valid {
		v := document.String
		c.Document = &v
	}
	if email.Valid {
		v := email.String
		c.Email = &v
	}
	if address.Valid {
		v := address.String
		c.Address = &v
	}
	if number.Valid {
		v := number.String
		c.Number = &v
	}
	if complement.Valid {
		v := complement.String
		c.Complement = &v
	}
	if district.Valid {
		v := district.String
		c.District = &v
	}
	if city.Valid {
		v := city.String
		c.City = &v
	}
	if state.Valid {
		v := state.String
		c.State = &v
	}
	if postalCode.Valid {
		v := postalCode.String
		c.PostalCode = &v
	}
	if externalID.Valid {
		v := externalID.String
		c.ExternalID = &v
	}
}
