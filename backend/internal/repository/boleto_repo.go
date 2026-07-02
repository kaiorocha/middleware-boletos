package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type BoletoRepo struct{ db *sql.DB }

func NewBoletoRepo(db *sql.DB) *BoletoRepo { return &BoletoRepo{db: db} }

func (r *BoletoRepo) Create(b *domain.Boleto) error {
	if b.ID == "" { b.ID = uuid.New().String() }
	_, err := r.db.Exec(`INSERT INTO boletos (id,tenant_id,customer_id,provider_id,amount_cents,due_date,status,external_id,barcode,digitable_line,our_number,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now(),now())`, b.ID,b.TenantID,b.CustomerID,b.ProviderID,b.AmountCents,b.DueDate,b.Status,b.ExternalID,b.Barcode,b.DigitableLine,b.OurNumber)
	return err
}

func (r *BoletoRepo) FindByID(id string) (*domain.Boleto, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,customer_id,provider_id,amount_cents,due_date,status,external_id,barcode,digitable_line,our_number,created_at,updated_at,deleted_at FROM boletos WHERE id = $1`, id)
	var b domain.Boleto
	var providerID sql.NullString
	var external sql.NullString
	var barcode sql.NullString
	var digitable sql.NullString
	var ourNumber sql.NullString
	var deleted *time.Time
	if err := row.Scan(&b.ID,&b.TenantID,&b.CustomerID,&providerID,&b.AmountCents,&b.DueDate,&b.Status,&external,&barcode,&digitable,&ourNumber,&b.CreatedAt,&b.UpdatedAt,&deleted); err != nil { return nil, err }
	if providerID.Valid { v := providerID.String; b.ProviderID = &v }
	if external.Valid { v := external.String; b.ExternalID = &v }
	if barcode.Valid { v := barcode.String; b.Barcode = &v }
	if digitable.Valid { v := digitable.String; b.DigitableLine = &v }
	if ourNumber.Valid { v := ourNumber.String; b.OurNumber = &v }
	if deleted != nil { b.DeletedAt = deleted }
	return &b, nil
}

func (r *BoletoRepo) ListByTenant(tenantID string) ([]domain.Boleto, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,customer_id,provider_id,amount_cents,due_date,status,external_id,barcode,digitable_line,our_number,created_at,updated_at,deleted_at FROM boletos WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []domain.Boleto
	for rows.Next() {
		var b domain.Boleto
		var providerID sql.NullString
		var external sql.NullString
		var barcode sql.NullString
		var digitable sql.NullString
		var ourNumber sql.NullString
		var deleted *time.Time
		if err := rows.Scan(&b.ID,&b.TenantID,&b.CustomerID,&providerID,&b.AmountCents,&b.DueDate,&b.Status,&external,&barcode,&digitable,&ourNumber,&b.CreatedAt,&b.UpdatedAt,&deleted); err != nil { return nil, err }
		if providerID.Valid { v := providerID.String; b.ProviderID = &v }
		if external.Valid { v := external.String; b.ExternalID = &v }
		if barcode.Valid { v := barcode.String; b.Barcode = &v }
		if digitable.Valid { v := digitable.String; b.DigitableLine = &v }
		if ourNumber.Valid { v := ourNumber.String; b.OurNumber = &v }
		if deleted != nil { b.DeletedAt = deleted }
		out = append(out, b)
	}
	return out, nil
}
