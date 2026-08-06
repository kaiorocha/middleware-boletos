package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type BlacklistRepo struct{ db *sql.DB }

func NewBlacklistRepo(db *sql.DB) *BlacklistRepo { return &BlacklistRepo{db: db} }

func (r *BlacklistRepo) Create(entry *domain.BlacklistEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	_, err := r.db.Exec(
		`INSERT INTO blacklist (id,tenant_id,document,name,reason,notes,source,created_by,active,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now(),now())`,
		entry.ID, entry.TenantID, entry.Document, entry.Name, entry.Reason, entry.Notes, entry.Source, entry.CreatedBy, entry.Active,
	)
	return translatePostgresError(err)
}

func (r *BlacklistRepo) FindByID(tenantID, id string) (*domain.BlacklistEntry, error) {
	row := r.db.QueryRow(
		`SELECT id,tenant_id,document,name,reason,notes,source,created_by,active,created_at,updated_at,deleted_at
		 FROM blacklist
		 WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id,
	)
	return scanBlacklistEntry(row)
}

func (r *BlacklistRepo) FindByDocument(tenantID, document string) (*domain.BlacklistEntry, error) {
	row := r.db.QueryRow(
		`SELECT id,tenant_id,document,name,reason,notes,source,created_by,active,created_at,updated_at,deleted_at
		 FROM blacklist
		 WHERE tenant_id = $1 AND document = $2 AND deleted_at IS NULL
		 ORDER BY active DESC, created_at DESC
		 LIMIT 1`,
		tenantID, document,
	)
	return scanBlacklistEntry(row)
}

func (r *BlacklistRepo) List(tenantID, search string, active *bool) ([]domain.BlacklistEntry, error) {
	var activeParam any
	if active != nil {
		activeParam = *active
	}
	query := `SELECT id,tenant_id,document,name,reason,notes,source,created_by,active,created_at,updated_at,deleted_at
		FROM blacklist
		WHERE tenant_id = $1 AND deleted_at IS NULL
		  AND ($2 = '' OR document ILIKE '%' || $2 || '%' OR name ILIKE '%' || $2 || '%')
		  AND ($3::boolean IS NULL OR active = $3)
		ORDER BY created_at DESC`
	rows, err := r.db.Query(query, tenantID, search, activeParam)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.BlacklistEntry
	for rows.Next() {
		entry, err := scanBlacklistEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *entry)
	}
	return out, rows.Err()
}

func (r *BlacklistRepo) Update(entry *domain.BlacklistEntry) error {
	_, err := r.db.Exec(
		`UPDATE blacklist
		 SET document = $1, name = $2, reason = $3, notes = $4, source = $5, created_by = $6, active = $7, updated_at = now()
		 WHERE tenant_id = $8 AND id = $9 AND deleted_at IS NULL`,
		entry.Document, entry.Name, entry.Reason, entry.Notes, entry.Source, entry.CreatedBy, entry.Active, entry.TenantID, entry.ID,
	)
	return translatePostgresError(err)
}

func (r *BlacklistRepo) SoftDelete(tenantID, id string) error {
	_, err := r.db.Exec(
		`UPDATE blacklist SET active = false, deleted_at = $1, updated_at = $1 WHERE tenant_id = $2 AND id = $3 AND deleted_at IS NULL`,
		time.Now().UTC(), tenantID, id,
	)
	return err
}

func (r *BlacklistRepo) IsBlocked(tenantID, document string) (*domain.BlacklistEntry, bool, error) {
	row := r.db.QueryRow(
		`SELECT id,tenant_id,document,name,reason,notes,source,created_by,active,created_at,updated_at,deleted_at
		 FROM blacklist
		 WHERE tenant_id = $1 AND document = $2 AND active = true AND deleted_at IS NULL
		 LIMIT 1`,
		tenantID, document,
	)
	entry, err := scanBlacklistEntry(row)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return entry, true, nil
}

type blacklistScanner interface {
	Scan(dest ...any) error
}

func scanBlacklistEntry(scanner blacklistScanner) (*domain.BlacklistEntry, error) {
	var entry domain.BlacklistEntry
	var notes sql.NullString
	var createdBy sql.NullString
	var deletedAt *time.Time
	if err := scanner.Scan(
		&entry.ID,
		&entry.TenantID,
		&entry.Document,
		&entry.Name,
		&entry.Reason,
		&notes,
		&entry.Source,
		&createdBy,
		&entry.Active,
		&entry.CreatedAt,
		&entry.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}
	if notes.Valid {
		v := notes.String
		entry.Notes = &v
	}
	if createdBy.Valid {
		v := createdBy.String
		entry.CreatedBy = &v
	}
	if deletedAt != nil {
		entry.DeletedAt = deletedAt
	}
	return &entry, nil
}
