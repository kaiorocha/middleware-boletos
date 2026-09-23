package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/lib/pq"
)

type CampaignRepo struct{ db *sql.DB }

func NewCampaignRepo(db *sql.DB) *CampaignRepo { return &CampaignRepo{db: db} }

func (r *CampaignRepo) Create(c *domain.Campaign) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return r.db.QueryRow(`INSERT INTO campaigns (id,tenant_id,name,description,status,created_by) VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at,updated_at`, c.ID, c.TenantID, c.Name, c.Description, c.Status, c.CreatedBy).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func scanCampaign(row interface{ Scan(...any) error }) (*domain.Campaign, error) {
	var c domain.Campaign
	var description, createdBy sql.NullString
	var started, completed sql.NullTime
	err := row.Scan(&c.ID, &c.TenantID, &c.Name, &description, &c.Status, &createdBy, &started, &completed, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if description.Valid {
		c.Description = &description.String
	}
	if createdBy.Valid {
		c.CreatedBy = &createdBy.String
	}
	if started.Valid {
		c.StartedAt = &started.Time
	}
	if completed.Valid {
		c.CompletedAt = &completed.Time
	}
	return &c, nil
}

const campaignColumns = `id,tenant_id,name,description,status,created_by,started_at,completed_at,created_at,updated_at`

func (r *CampaignRepo) Get(tenantID, id string) (*domain.Campaign, error) {
	return scanCampaign(r.db.QueryRow(`SELECT `+campaignColumns+` FROM campaigns WHERE tenant_id=$1 AND id=$2`, tenantID, id))
}

func (r *CampaignRepo) Update(c *domain.Campaign) error {
	return r.db.QueryRow(`UPDATE campaigns SET name=$1,description=$2,updated_at=now() WHERE id=$3 AND tenant_id=$4 AND status='DRAFT' RETURNING updated_at`, c.Name, c.Description, c.ID, c.TenantID).Scan(&c.UpdatedAt)
}

func (r *CampaignRepo) List(f domain.CampaignFilters) ([]domain.Campaign, int, error) {
	where := []string{"tenant_id=$1"}
	args := []any{f.TenantID}
	add := func(condition string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(condition, len(args)))
	}
	if f.Status != "" {
		add("status=$%d", f.Status)
	}
	if f.Search != "" {
		add("name ILIKE '%%'||$%d||'%%'", f.Search)
	}
	if f.From != nil {
		add("created_at >= $%d", *f.From)
	}
	if f.To != nil {
		add("created_at < $%d", f.To.Add(24*time.Hour))
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var total int
	if err := r.db.QueryRow(`SELECT count(*) FROM campaigns`+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(`SELECT `+campaignColumns+` FROM campaigns`+w+fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.Campaign{}
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

func (r *CampaignRepo) SavePreview(campaignID, tenantID, checksum string, rows []domain.StagedCampaignRow, p domain.CampaignImportPreview) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	id := uuid.NewString()
	_, err = tx.Exec(`INSERT INTO campaign_imports(id,campaign_id,tenant_id,checksum,total_rows,valid_rows,invalid_rows,total_amount_cents,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,now()+interval '24 hours') ON CONFLICT(campaign_id,checksum) DO UPDATE SET expires_at=excluded.expires_at`, id, campaignID, tenantID, checksum, p.TotalRows, p.ValidRows, p.InvalidRows, p.TotalAmountCents)
	if err != nil {
		return "", err
	}
	var actual string
	if err = tx.QueryRow(`SELECT id FROM campaign_imports WHERE campaign_id=$1 AND checksum=$2`, campaignID, checksum).Scan(&actual); err != nil {
		return "", err
	}
	if actual == id {
		stmt, err := tx.Prepare(`INSERT INTO campaign_import_rows(import_id,row_number,recipient_email,payer_document,amount_cents,due_date,external_id,field,error_code,error_message) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`)
		if err != nil {
			return "", err
		}
		defer stmt.Close()
		for _, row := range rows {
			var due any
			if !row.DueDate.IsZero() {
				due = row.DueDate
			}
			if _, err = stmt.Exec(actual, row.Row, nullableString(row.Email), nullableString(row.Document), nullableInt64(row.Amount), due, row.ExternalID, nullableString(row.Field), nullableString(row.Code), nullableString(row.Message)); err != nil {
				return "", err
			}
		}
	}
	return actual, tx.Commit()
}

func nullableInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func (r *CampaignRepo) ExistingExternalIDs(tenantID string, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(`SELECT external_id FROM boletos WHERE tenant_id=$1 AND external_id=ANY($2) AND deleted_at IS NULL`, tenantID, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (r *CampaignRepo) ConfirmImport(tenantID, campaignID, importID, providerID string) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var status string
	var validRows int
	if err = tx.QueryRow(`SELECT status,valid_rows FROM campaign_imports WHERE id=$1 AND campaign_id=$2 AND tenant_id=$3 AND expires_at>now() FOR UPDATE`, importID, campaignID, tenantID).Scan(&status, &validRows); err != nil {
		return 0, err
	}
	if status != "PREVIEWED" || validRows == 0 {
		return 0, fmt.Errorf("import is not previewed")
	}
	res, err := tx.Exec(`INSERT INTO boletos(id,tenant_id,campaign_id,recipient_email,payer_document,provider_id,amount_cents,due_date,status,external_id,created_at,updated_at) SELECT gen_random_uuid(),$1,$2,recipient_email,payer_document,$3,amount_cents,due_date,'CREATED',external_id,now(),now() FROM campaign_import_rows WHERE import_id=$4 AND error_code IS NULL ORDER BY row_number`, tenantID, campaignID, providerID, importID)
	if err != nil {
		return 0, translatePostgresError(err)
	}
	n, _ := res.RowsAffected()
	if _, err = tx.Exec(`UPDATE campaign_imports SET status='CONFIRMED' WHERE id=$1`, importID); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`UPDATE campaigns SET status='READY',updated_at=now() WHERE id=$1 AND tenant_id=$2 AND status='DRAFT'`, campaignID, tenantID); err != nil {
		return 0, err
	}
	return int(n), tx.Commit()
}

func (r *CampaignRepo) Start(tenantID, id string) error {
	res, err := r.db.Exec(`UPDATE campaigns SET status='PROCESSING',started_at=now(),updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='READY'`, tenantID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *CampaignRepo) ProcessingCampaigns(limit int) ([]domain.Campaign, error) {
	rows, err := r.db.Query(`SELECT `+campaignColumns+` FROM campaigns WHERE status='PROCESSING' ORDER BY started_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Campaign{}
	for rows.Next() {
		c, e := scanCampaign(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}
func (r *CampaignRepo) ClaimPendingBoletoIDs(campaignID string, limit int) ([]string, error) {
	rows, err := r.db.Query(`WITH next AS (
		SELECT id FROM boletos WHERE campaign_id=$1 AND deleted_at IS NULL
		AND (status='CREATED' OR (status='PROCESSING' AND campaign_claimed_at < now()-interval '5 minutes'))
		ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $2
	) UPDATE boletos b SET status='PROCESSING',campaign_claimed_at=now(),updated_at=now()
	FROM next WHERE b.id=next.id RETURNING b.id`, campaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func (r *CampaignRepo) FinishIfDone(campaignID string) error {
	_, err := r.db.Exec(`UPDATE campaigns c SET status=CASE WHEN EXISTS(SELECT 1 FROM boletos b WHERE b.campaign_id=c.id AND b.status IN('FAILED','BLOCKED')) THEN 'PARTIAL' ELSE 'COMPLETED' END,completed_at=now(),updated_at=now() WHERE c.id=$1 AND c.status='PROCESSING' AND NOT EXISTS(SELECT 1 FROM boletos b WHERE b.campaign_id=c.id AND b.status IN('CREATED','PROCESSING'))`, campaignID)
	return err
}

func (r *CampaignRepo) MarkBoletoTerminal(id, status string) error {
	_, err := r.db.Exec(`UPDATE boletos SET status=$1,campaign_claimed_at=NULL,updated_at=now() WHERE id=$2 AND status IN ('CREATED','PROCESSING')`, status, id)
	return err
}

func (r *CampaignRepo) Metrics(tenantID, campaignID string) (*domain.CampaignMetrics, error) {
	m := &domain.CampaignMetrics{}
	err := r.db.QueryRow(`SELECT count(*),count(*) FILTER(WHERE status='CREATED'),count(*) FILTER(WHERE status='PROCESSING'),count(*) FILTER(WHERE status IN('ISSUED','PAID','EXPIRED','CANCELLED')),count(*) FILTER(WHERE status='PAID'),count(*) FILTER(WHERE status='FAILED'),count(*) FILTER(WHERE status='BLOCKED'),count(*) FILTER(WHERE status='CANCELLED'),coalesce(sum(amount_cents) FILTER(WHERE status IN('ISSUED','PAID','EXPIRED','CANCELLED')),0),coalesce(sum(amount_cents) FILTER(WHERE status='PAID'),0) FROM boletos WHERE tenant_id=$1 AND campaign_id=$2 AND deleted_at IS NULL`, tenantID, campaignID).Scan(&m.TotalImported, &m.Waiting, &m.Processing, &m.Issued, &m.Paid, &m.Failed, &m.Blocked, &m.Cancelled, &m.IssuedAmountCents, &m.PaidAmountCents)
	if err != nil {
		return nil, err
	}
	processed := m.TotalImported - m.Waiting - m.Processing
	if m.TotalImported > 0 {
		m.ProcessedPercent = float64(processed) / float64(m.TotalImported) * 100
	}
	if m.Issued > 0 {
		m.Conversion = float64(m.Paid) / float64(m.Issued) * 100
	}
	if m.Paid > 0 {
		m.AverageTicketCents = m.PaidAmountCents / int64(m.Paid)
	}
	m.ByStatus, err = r.metric(`SELECT status,status,count(*),coalesce(sum(amount_cents),0) FROM boletos WHERE tenant_id=$1 AND campaign_id=$2 AND deleted_at IS NULL GROUP BY status ORDER BY count(*) DESC`, tenantID, campaignID)
	if err != nil {
		return nil, err
	}
	m.ByDueDate, err = r.metric(`SELECT due_date::text,due_date::text,count(*) FILTER(WHERE status IN('ISSUED','PAID','EXPIRED','CANCELLED')),coalesce(sum(amount_cents) FILTER(WHERE status='PAID'),0) FROM boletos WHERE tenant_id=$1 AND campaign_id=$2 AND deleted_at IS NULL GROUP BY due_date ORDER BY due_date`, tenantID, campaignID)
	if err != nil {
		return nil, err
	}
	m.ByAmount, err = r.metric(`SELECT amount_cents::text,amount_cents::text,count(*) FILTER(WHERE status IN('ISSUED','PAID','EXPIRED','CANCELLED')),coalesce(sum(amount_cents) FILTER(WHERE status='PAID'),0) FROM boletos WHERE tenant_id=$1 AND campaign_id=$2 AND deleted_at IS NULL GROUP BY amount_cents ORDER BY amount_cents`, tenantID, campaignID)
	return m, err
}

func (r *CampaignRepo) Dashboard(tenantID, campaignID, providerID, status string, from, to *time.Time) (*domain.CampaignDashboard, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(q string, v any) { args = append(args, v); where = append(where, fmt.Sprintf(q, len(args))) }
	if tenantID != "" {
		add("c.tenant_id=$%d", tenantID)
	}
	if campaignID != "" {
		add("c.id=$%d", campaignID)
	}
	if providerID != "" {
		add("b.provider_id=$%d", providerID)
	}
	if status != "" {
		add("c.status=$%d", status)
	}
	if from != nil {
		add("c.created_at >= $%d", *from)
	}
	if to != nil {
		add("c.created_at < $%d", to.Add(24*time.Hour))
	}
	w := strings.Join(where, " AND ")
	rows, err := r.db.Query(`SELECT c.id::text,c.name,c.tenant_id::text,count(b.id) FILTER(WHERE b.status IN('ISSUED','PAID','EXPIRED','CANCELLED')),count(b.id) FILTER(WHERE b.status='PAID'),coalesce(sum(b.amount_cents) FILTER(WHERE b.status='PAID'),0) FROM campaigns c LEFT JOIN boletos b ON b.campaign_id=c.id AND b.deleted_at IS NULL WHERE `+w+` GROUP BY c.id,c.name,c.tenant_id ORDER BY count(b.id) DESC,c.created_at DESC LIMIT 500`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d := &domain.CampaignDashboard{Performance: []domain.CampaignPerformance{}}
	for rows.Next() {
		var p domain.CampaignPerformance
		if err := rows.Scan(&p.CampaignID, &p.CampaignName, &p.TenantID, &p.Issued, &p.Paid, &p.RevenueCents); err != nil {
			return nil, err
		}
		if p.Issued > 0 {
			p.Conversion = float64(p.Paid) / float64(p.Issued) * 100
		}
		if p.Paid > 0 {
			p.AverageTicketCents = p.RevenueCents / int64(p.Paid)
		}
		d.Performance = append(d.Performance, p)
		d.Campaigns++
		d.Issued += p.Issued
		d.Paid += p.Paid
		d.RevenueCents += p.RevenueCents
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	err = r.db.QueryRow(`SELECT coalesce(sum(b.amount_cents) FILTER(WHERE b.status IN('ISSUED','PAID','EXPIRED','CANCELLED')),0) FROM campaigns c LEFT JOIN boletos b ON b.campaign_id=c.id AND b.deleted_at IS NULL WHERE `+w, args...).Scan(&d.IssuedAmountCents)
	if err != nil {
		return nil, err
	}
	if d.Issued > 0 {
		d.Conversion = float64(d.Paid) / float64(d.Issued) * 100
	}
	if d.Paid > 0 {
		d.AverageTicketCents = d.RevenueCents / int64(d.Paid)
	}
	return d, nil
}
func (r *CampaignRepo) metric(q string, args ...any) ([]domain.MetricRow, error) {
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.MetricRow{}
	for rows.Next() {
		var x domain.MetricRow
		if err := rows.Scan(&x.ID, &x.Label, &x.Count, &x.AmountCents); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
