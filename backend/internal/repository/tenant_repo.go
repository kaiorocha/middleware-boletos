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
	_, err := r.db.Exec(`INSERT INTO tenants (id,name,document,address,district,city,postal_code,state,country_code,area_code,phone_number,webhook_url,owner_id,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,now(),now())`, t.ID, t.Name, t.Document, t.Address, t.District, t.City, t.PostalCode, t.State, t.CountryCode, t.AreaCode, t.PhoneNumber, t.WebhookURL, t.OwnerID)
	return err
}

func (r *TenantRepo) FindByID(id string) (*domain.Tenant, error) {
	row := r.db.QueryRow(`SELECT id,name,COALESCE(document,''),COALESCE(address,''),COALESCE(district,''),COALESCE(city,''),COALESCE(postal_code,''),COALESCE(state,''),COALESCE(country_code,''),COALESCE(area_code,''),COALESCE(phone_number,''),COALESCE(webhook_url,''),owner_id,created_at,updated_at,deleted_at FROM tenants WHERE id = $1 AND deleted_at IS NULL`, id)
	var t domain.Tenant
	var ownerID sql.NullString
	var deleted *time.Time
	if err := row.Scan(&t.ID, &t.Name, &t.Document, &t.Address, &t.District, &t.City, &t.PostalCode, &t.State, &t.CountryCode, &t.AreaCode, &t.PhoneNumber, &t.WebhookURL, &ownerID, &t.CreatedAt, &t.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	if ownerID.Valid {
		v := ownerID.String
		t.OwnerID = &v
	}
	if deleted != nil {
		t.DeletedAt = deleted
	}
	return &t, nil
}

func (r *TenantRepo) List() ([]domain.Tenant, error) {
	rows, err := r.db.Query(`SELECT id,name,COALESCE(document,''),COALESCE(address,''),COALESCE(district,''),COALESCE(city,''),COALESCE(postal_code,''),COALESCE(state,''),COALESCE(country_code,''),COALESCE(area_code,''),COALESCE(phone_number,''),COALESCE(webhook_url,''),owner_id,created_at,updated_at,deleted_at FROM tenants WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		var ownerID sql.NullString
		var deleted *time.Time
		if err := rows.Scan(&t.ID, &t.Name, &t.Document, &t.Address, &t.District, &t.City, &t.PostalCode, &t.State, &t.CountryCode, &t.AreaCode, &t.PhoneNumber, &t.WebhookURL, &ownerID, &t.CreatedAt, &t.UpdatedAt, &deleted); err != nil {
			return nil, err
		}
		if ownerID.Valid {
			v := ownerID.String
			t.OwnerID = &v
		}
		if deleted != nil {
			t.DeletedAt = deleted
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *TenantRepo) Update(t *domain.Tenant) error {
	_, err := r.db.Exec(`UPDATE tenants SET name=$1,document=$2,address=$3,district=$4,city=$5,postal_code=$6,state=$7,country_code=$8,area_code=$9,phone_number=$10,webhook_url=$11,owner_id=$12,updated_at=now() WHERE id=$13 AND deleted_at IS NULL`, t.Name, t.Document, t.Address, t.District, t.City, t.PostalCode, t.State, t.CountryCode, t.AreaCode, t.PhoneNumber, t.WebhookURL, t.OwnerID, t.ID)
	return err
}

func (r *TenantRepo) Delete(id string) error {
	_, err := r.db.Exec(`UPDATE tenants SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL`, time.Now().UTC(), id)
	return err
}
