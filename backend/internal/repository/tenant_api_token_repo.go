package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type TenantAPITokenRepo struct{ db *sql.DB }

func NewTenantAPITokenRepo(db *sql.DB) *TenantAPITokenRepo { return &TenantAPITokenRepo{db: db} }

func (r *TenantAPITokenRepo) Rotate(token *domain.TenantAPIToken, tokenHash string) error {
	return r.RotateSecure(token, tokenHash, "")
}

func (r *TenantAPITokenRepo) RotateSecure(token *domain.TenantAPIToken, tokenHash, encryptedToken string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if _, err = tx.Exec(`UPDATE tenant_api_tokens SET status='REVOKED', revoked_at=$1 WHERE tenant_id=$2 AND environment=$3 AND status='ACTIVE'`, now, token.TenantID, token.Environment); err != nil {
		return err
	}
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	if err = tx.QueryRow(`INSERT INTO tenant_api_tokens (id,tenant_id,environment,token_hash,token_prefix,encrypted_token,status,created_at) VALUES ($1,$2,$3,$4,$5,$6,'ACTIVE',now()) RETURNING created_at`, token.ID, token.TenantID, token.Environment, tokenHash, token.TokenPrefix, nullableString(encryptedToken)).Scan(&token.CreatedAt); err != nil {
		return err
	}
	token.Status = "ACTIVE"
	return tx.Commit()
}

func (r *TenantAPITokenRepo) ListActiveByTenant(tenantID string) ([]domain.TenantAPIToken, error) {
	rows, err := r.db.Query(`SELECT id,tenant_id,environment,token_prefix,status,created_at,revoked_at,COALESCE(encrypted_token,'') FROM tenant_api_tokens WHERE tenant_id=$1 AND status='ACTIVE' ORDER BY environment`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []domain.TenantAPIToken
	for rows.Next() {
		var token domain.TenantAPIToken
		if err := rows.Scan(&token.ID, &token.TenantID, &token.Environment, &token.TokenPrefix, &token.Status, &token.CreatedAt, &token.RevokedAt, &token.EncryptedToken); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (r *TenantAPITokenRepo) FindActiveByTenantEnvironment(tenantID, environment string) (*domain.TenantAPIToken, error) {
	var token domain.TenantAPIToken
	err := r.db.QueryRow(`SELECT id,tenant_id,environment,token_prefix,status,created_at,revoked_at,COALESCE(encrypted_token,'') FROM tenant_api_tokens WHERE tenant_id=$1 AND environment=$2 AND status='ACTIVE'`, tenantID, environment).Scan(&token.ID, &token.TenantID, &token.Environment, &token.TokenPrefix, &token.Status, &token.CreatedAt, &token.RevokedAt, &token.EncryptedToken)
	return &token, err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *TenantAPITokenRepo) FindActiveByHash(tokenHash string) (*domain.TenantAPIToken, error) {
	var token domain.TenantAPIToken
	err := r.db.QueryRow(`SELECT id,tenant_id,environment,token_prefix,status,created_at,revoked_at FROM tenant_api_tokens WHERE token_hash=$1 AND status='ACTIVE'`, tokenHash).Scan(&token.ID, &token.TenantID, &token.Environment, &token.TokenPrefix, &token.Status, &token.CreatedAt, &token.RevokedAt)
	return &token, err
}
