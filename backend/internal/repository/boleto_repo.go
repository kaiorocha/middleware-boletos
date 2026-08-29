package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type BoletoRepo struct{ db *sql.DB }

func NewBoletoRepo(db *sql.DB) *BoletoRepo { return &BoletoRepo{db: db} }

func (r *BoletoRepo) Create(b *domain.Boleto) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	_, err := r.db.Exec(`INSERT INTO boletos (id,tenant_id,customer_id,recipient_email,provider_id,amount_cents,due_date,status,external_id,barcode,digitable_line,our_number,base64,issued_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now(),now())`, b.ID, b.TenantID, b.CustomerID, b.RecipientEmail, b.ProviderID, b.AmountCents, b.DueDate, b.Status, b.ExternalID, b.Barcode, b.DigitableLine, b.OurNumber, b.Base64, b.IssuedAt)
	return translatePostgresError(err)
}

func (r *BoletoRepo) FindByID(id string) (*domain.Boleto, error) {
	row := r.db.QueryRow(`SELECT id,tenant_id,customer_id,recipient_email,provider_id,amount_cents,due_date,status,external_id,barcode,digitable_line,our_number,base64,issued_at,created_at,updated_at,deleted_at FROM boletos WHERE id = $1 AND deleted_at IS NULL`, id)
	var b domain.Boleto
	var customerID sql.NullString
	var recipientEmail sql.NullString
	var providerID sql.NullString
	var external sql.NullString
	var barcode sql.NullString
	var digitable sql.NullString
	var ourNumber sql.NullString
	var base64Value sql.NullString
	var issuedAt sql.NullTime
	var deleted *time.Time
	if err := row.Scan(&b.ID, &b.TenantID, &customerID, &recipientEmail, &providerID, &b.AmountCents, &b.DueDate, &b.Status, &external, &barcode, &digitable, &ourNumber, &base64Value, &issuedAt, &b.CreatedAt, &b.UpdatedAt, &deleted); err != nil {
		return nil, err
	}
	if customerID.Valid {
		b.CustomerID = &customerID.String
	}
	if recipientEmail.Valid {
		b.RecipientEmail = recipientEmail.String
	}
	if providerID.Valid {
		v := providerID.String
		b.ProviderID = &v
	}
	if external.Valid {
		v := external.String
		b.ExternalID = &v
	}
	if barcode.Valid {
		v := barcode.String
		b.Barcode = &v
	}
	if digitable.Valid {
		v := digitable.String
		b.DigitableLine = &v
	}
	if ourNumber.Valid {
		v := ourNumber.String
		b.OurNumber = &v
	}
	if base64Value.Valid {
		v := base64Value.String
		b.Base64 = &v
	}
	if issuedAt.Valid {
		v := issuedAt.Time
		b.IssuedAt = &v
	}
	if deleted != nil {
		b.DeletedAt = deleted
	}
	return &b, nil
}

func (r *BoletoRepo) FindByProviderReference(providerID, customerReference, ourNumber string) (*domain.Boleto, error) {
	var id string
	err := r.db.QueryRow(`SELECT id FROM boletos
		WHERE provider_id=$1 AND deleted_at IS NULL
		AND (($2 <> '' AND (id::text=$2 OR external_id=$2)) OR ($3 <> '' AND our_number=$3))
		ORDER BY CASE WHEN $2 <> '' AND (id::text=$2 OR external_id=$2) THEN 0 ELSE 1 END LIMIT 1`,
		providerID, customerReference, ourNumber).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.FindByID(id)
}

func (r *BoletoRepo) ListForProviderSync(limit int) ([]domain.Boleto, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Query(`SELECT id FROM boletos WHERE deleted_at IS NULL AND our_number IS NOT NULL AND our_number <> '' AND status IN ('PROCESSING','ISSUED','PARTIAL') ORDER BY COALESCE(last_provider_sync_at,updated_at) ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]domain.Boleto, 0, len(ids))
	for _, id := range ids {
		item, err := r.FindByID(id)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (r *BoletoRepo) MarkProviderSynced(id string) error {
	_, err := r.db.Exec(`UPDATE boletos SET last_provider_sync_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	return err
}

func (r *BoletoRepo) ListByTenant(tenantID string) ([]domain.Boleto, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,customer_id,recipient_email,provider_id,amount_cents,due_date,status,external_id,barcode,digitable_line,our_number,base64,issued_at,created_at,updated_at,deleted_at FROM boletos WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Boleto
	for rows.Next() {
		var b domain.Boleto
		var customerID sql.NullString
		var recipientEmail sql.NullString
		var providerID sql.NullString
		var external sql.NullString
		var barcode sql.NullString
		var digitable sql.NullString
		var ourNumber sql.NullString
		var base64Value sql.NullString
		var issuedAt sql.NullTime
		var deleted *time.Time
		if err := rows.Scan(&b.ID, &b.TenantID, &customerID, &recipientEmail, &providerID, &b.AmountCents, &b.DueDate, &b.Status, &external, &barcode, &digitable, &ourNumber, &base64Value, &issuedAt, &b.CreatedAt, &b.UpdatedAt, &deleted); err != nil {
			return nil, err
		}
		if customerID.Valid {
			b.CustomerID = &customerID.String
		}
		if recipientEmail.Valid {
			b.RecipientEmail = recipientEmail.String
		}
		if providerID.Valid {
			v := providerID.String
			b.ProviderID = &v
		}
		if external.Valid {
			v := external.String
			b.ExternalID = &v
		}
		if barcode.Valid {
			v := barcode.String
			b.Barcode = &v
		}
		if digitable.Valid {
			v := digitable.String
			b.DigitableLine = &v
		}
		if ourNumber.Valid {
			v := ourNumber.String
			b.OurNumber = &v
		}
		if base64Value.Valid {
			v := base64Value.String
			b.Base64 = &v
		}
		if issuedAt.Valid {
			v := issuedAt.Time
			b.IssuedAt = &v
		}
		if deleted != nil {
			b.DeletedAt = deleted
		}
		out = append(out, b)
	}
	return out, nil
}

func (r *BoletoRepo) Update(b *domain.Boleto) error {
	_, err := r.db.Exec(`UPDATE boletos SET provider_id=$1,amount_cents=$2,due_date=$3,status=$4,external_id=$5,barcode=$6,digitable_line=$7,our_number=$8,base64=$9,issued_at=$10,customer_id=$11,recipient_email=$12,updated_at=now() WHERE id=$13 AND tenant_id=$14 AND deleted_at IS NULL`, b.ProviderID, b.AmountCents, b.DueDate, b.Status, b.ExternalID, b.Barcode, b.DigitableLine, b.OurNumber, b.Base64, b.IssuedAt, b.CustomerID, b.RecipientEmail, b.ID, b.TenantID)
	return translatePostgresError(err)
}

func (r *BoletoRepo) Delete(id string, tenantID string) error {
	_, err := r.db.Exec(`UPDATE boletos SET deleted_at = $1, updated_at = $1, status = 'INACTIVE' WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`, time.Now().UTC(), id, tenantID)
	return err
}

func (r *BoletoRepo) AdminDashboard(filters domain.BoletoFilters) (*domain.AdminDashboard, error) {
	where, args := adminBoletoWhere(filters)
	var dash domain.AdminDashboard
	var totalAmount sql.NullInt64
	err := r.db.QueryRow(`
		SELECT
			COUNT(DISTINCT t.id),
			COUNT(b.id),
			COALESCE(SUM(b.amount_cents) FILTER (WHERE b.status IN ('ISSUED','PAID','EXPIRED','CANCELLED')), 0),
			COUNT(*) FILTER (WHERE b.status = 'ISSUED'),
			COUNT(*) FILTER (WHERE b.status = 'PAID'),
			COUNT(*) FILTER (WHERE b.status = 'FAILED'),
			COUNT(*) FILTER (WHERE b.status = 'CREATED'),
			COUNT(*) FILTER (WHERE b.status = 'PROCESSING'),
			COUNT(*) FILTER (WHERE b.status = 'EXPIRED'),
			COUNT(*) FILTER (WHERE b.status = 'CANCELLED')
		FROM tenants t
		LEFT JOIN boletos b ON b.tenant_id = t.id AND b.deleted_at IS NULL
		LEFT JOIN customers c ON c.id = b.customer_id
		LEFT JOIN providers p ON p.id = b.provider_id
		`+where, args...).Scan(
		&dash.Totals.Tenants,
		&dash.Totals.Boletos,
		&totalAmount,
		&dash.Totals.Issued,
		&dash.Totals.Paid,
		&dash.Totals.Failed,
		&dash.Totals.Created,
		&dash.Totals.Processing,
		&dash.Totals.Expired,
		&dash.Totals.Cancelled,
	)
	if err != nil {
		return nil, err
	}
	dash.Totals.AmountCents = totalAmount.Int64
	successful := dash.Totals.Issued + dash.Totals.Paid + dash.Totals.Expired + dash.Totals.Cancelled
	completedAttempts := successful + dash.Totals.Failed
	if completedAttempts > 0 {
		dash.Totals.SuccessRate = float64(successful) / float64(completedAttempts)
		dash.Totals.FailureRate = float64(dash.Totals.Failed) / float64(completedAttempts)
	}
	if successful > 0 {
		dash.Totals.AverageTicketCents = dash.Totals.AmountCents / int64(successful)
	}

	if dash.ByTenant, err = r.metricRows(`SELECT t.id::text, t.name, COUNT(b.id), COALESCE(SUM(b.amount_cents) FILTER (WHERE b.status IN ('ISSUED','PAID','EXPIRED','CANCELLED')),0) FROM tenants t LEFT JOIN boletos b ON b.tenant_id = t.id AND b.deleted_at IS NULL LEFT JOIN customers c ON c.id = b.customer_id LEFT JOIN providers p ON p.id = b.provider_id `+where+` GROUP BY t.id, t.name ORDER BY COUNT(b.id) DESC, t.name ASC LIMIT 10`, args...); err != nil {
		return nil, err
	}
	if dash.ByProvider, err = r.metricRows(`SELECT COALESCE(p.id::text,''), COALESCE(p.name,'Sem provider'), COUNT(b.id), COALESCE(SUM(b.amount_cents) FILTER (WHERE b.status IN ('ISSUED','PAID','EXPIRED','CANCELLED')),0) FROM boletos b LEFT JOIN customers c ON c.id = b.customer_id LEFT JOIN providers p ON p.id = b.provider_id LEFT JOIN tenants t ON t.id = b.tenant_id `+where+` GROUP BY p.id, p.name ORDER BY COUNT(b.id) DESC, COALESCE(p.name,'Sem provider') ASC LIMIT 10`, args...); err != nil {
		return nil, err
	}
	if dash.ByStatus, err = r.metricRows(`SELECT COALESCE(b.status,''), COALESCE(b.status,'Sem status'), COUNT(b.id), COALESCE(SUM(b.amount_cents) FILTER (WHERE b.status IN ('ISSUED','PAID','EXPIRED','CANCELLED')),0) FROM boletos b LEFT JOIN customers c ON c.id = b.customer_id LEFT JOIN providers p ON p.id = b.provider_id LEFT JOIN tenants t ON t.id = b.tenant_id `+where+` GROUP BY b.status ORDER BY COUNT(b.id) DESC`, args...); err != nil {
		return nil, err
	}
	if dash.Timeline, err = r.timelineRows(`SELECT to_char(date_trunc('day', b.created_at), 'YYYY-MM-DD'), COUNT(b.id), COALESCE(SUM(b.amount_cents) FILTER (WHERE b.status IN ('ISSUED','PAID','EXPIRED','CANCELLED')),0) FROM boletos b LEFT JOIN customers c ON c.id = b.customer_id LEFT JOIN providers p ON p.id = b.provider_id LEFT JOIN tenants t ON t.id = b.tenant_id `+where+` GROUP BY date_trunc('day', b.created_at) ORDER BY date_trunc('day', b.created_at) ASC`, args...); err != nil {
		return nil, err
	}
	return &dash, nil
}

func (r *BoletoRepo) ListTransactions(filters domain.BoletoFilters) (*domain.PaginatedTransactions, error) {
	if filters.Limit <= 0 || filters.Limit > 200 {
		filters.Limit = 50
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}
	where, args := adminBoletoWhere(filters)
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(b.id) FROM boletos b LEFT JOIN customers c ON c.id = b.customer_id LEFT JOIN providers p ON p.id = b.provider_id LEFT JOIN tenants t ON t.id = b.tenant_id `+where, args...).Scan(&total); err != nil {
		return nil, err
	}
	args = append(args, filters.Limit, filters.Offset)
	query := fmt.Sprintf(`
		SELECT b.id, b.tenant_id, COALESCE(t.name,''), b.customer_id, COALESCE(c.name,''), COALESCE(c.document,''), b.recipient_email, b.provider_id, p.name, b.amount_cents, b.due_date, b.status, b.external_id, b.our_number, b.created_at, b.issued_at, b.digitable_line, b.barcode, COALESCE(length(b.base64), 0)
		FROM boletos b
		LEFT JOIN tenants t ON t.id = b.tenant_id
		LEFT JOIN customers c ON c.id = b.customer_id
		LEFT JOIN providers p ON p.id = b.provider_id
		%s
		ORDER BY b.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args))
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := domain.PaginatedTransactions{Items: []domain.BoletoTransaction{}, Limit: filters.Limit, Offset: filters.Offset, Total: total}
	for rows.Next() {
		var item domain.BoletoTransaction
		var customerID sql.NullString
		var customerName sql.NullString
		var customerDocument sql.NullString
		var recipientEmail sql.NullString
		var providerID sql.NullString
		var providerName sql.NullString
		var externalID sql.NullString
		var ourNumber sql.NullString
		var issuedAt sql.NullTime
		var digitableLine sql.NullString
		var barcode sql.NullString
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TenantName, &customerID, &customerName, &customerDocument, &recipientEmail, &providerID, &providerName, &item.AmountCents, &item.DueDate, &item.Status, &externalID, &ourNumber, &item.CreatedAt, &issuedAt, &digitableLine, &barcode, &item.Base64Size); err != nil {
			return nil, err
		}
		if customerID.Valid {
			item.CustomerID = &customerID.String
		}
		if customerName.Valid {
			item.CustomerName = &customerName.String
		}
		if customerDocument.Valid {
			item.CustomerDocument = &customerDocument.String
		}
		if recipientEmail.Valid {
			item.RecipientEmail = recipientEmail.String
		}
		if providerID.Valid {
			v := providerID.String
			item.ProviderID = &v
		}
		if providerName.Valid {
			v := providerName.String
			item.ProviderName = &v
		}
		if externalID.Valid {
			v := externalID.String
			item.ExternalID = &v
		}
		if ourNumber.Valid {
			v := ourNumber.String
			item.OurNumber = &v
		}
		if issuedAt.Valid {
			v := issuedAt.Time
			item.IssuedAt = &v
		}
		if digitableLine.Valid {
			v := digitableLine.String
			item.DigitableLine = &v
		}
		if barcode.Valid {
			v := barcode.String
			item.Barcode = &v
		}
		item.Base64Available = item.Base64Size > 0
		out.Items = append(out.Items, item)
	}
	return &out, rows.Err()
}

func adminBoletoWhere(filters domain.BoletoFilters) (string, []any) {
	clauses := []string{"t.deleted_at IS NULL", "b.deleted_at IS NULL"}
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(sql, len(args)))
	}
	if filters.From != nil {
		add("b.created_at >= $%d", *filters.From)
	}
	if filters.To != nil {
		add("b.created_at < $%d", filters.To.Add(24*time.Hour))
	}
	if filters.TenantID != "" {
		add("t.id = $%d", filters.TenantID)
	}
	if filters.ProviderID != "" {
		add("b.provider_id = $%d", filters.ProviderID)
	}
	if filters.Status != "" {
		add("b.status = $%d", strings.ToUpper(strings.TrimSpace(filters.Status)))
	}
	if filters.ExternalID != "" {
		add("b.external_id = $%d", strings.TrimSpace(filters.ExternalID))
	}
	if filters.OurNumber != "" {
		add("b.our_number = $%d", strings.TrimSpace(filters.OurNumber))
	}
	if filters.Document != "" {
		add("c.document = $%d", normalizeDigits(filters.Document))
	}
	if filters.Email != "" {
		email := strings.ToLower(strings.TrimSpace(filters.Email))
		args = append(args, email, email)
		clauses = append(clauses, fmt.Sprintf("(b.recipient_email = $%d OR c.email = $%d)", len(args)-1, len(args)))
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func normalizeDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (r *BoletoRepo) metricRows(query string, args ...any) ([]domain.MetricRow, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.MetricRow{}
	for rows.Next() {
		var row domain.MetricRow
		if err := rows.Scan(&row.ID, &row.Label, &row.Count, &row.AmountCents); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *BoletoRepo) timelineRows(query string, args ...any) ([]domain.TimelineRow, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TimelineRow{}
	for rows.Next() {
		var row domain.TimelineRow
		if err := rows.Scan(&row.Date, &row.Count, &row.AmountCents); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
