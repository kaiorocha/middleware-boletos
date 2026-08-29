package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/contracts"
	providertypes "github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

type providerSyncBoletoRepo interface {
	FindByID(string) (*domain.Boleto, error)
	ListForProviderSync(int) ([]domain.Boleto, error)
	Update(*domain.Boleto) error
	MarkProviderSynced(string) error
}
type ProviderSyncResult struct {
	Boleto         *domain.Boleto `json:"boleto"`
	Updated        bool           `json:"updated"`
	TenantNotified bool           `json:"tenant_notified"`
}
type ProviderSyncService struct {
	db         *sql.DB
	boletos    providerSyncBoletoRepo
	tenants    moncalieriTenantRepo
	providers  moncalieriProviderRepo
	factory    contracts.ProviderFactory
	httpClient *http.Client
	logger     *slog.Logger
}

func NewProviderSyncService(db *sql.DB, boletos providerSyncBoletoRepo, tenants moncalieriTenantRepo, providers moncalieriProviderRepo, factory contracts.ProviderFactory) *ProviderSyncService {
	return &ProviderSyncService{db: db, boletos: boletos, tenants: tenants, providers: providers, factory: factory, httpClient: &http.Client{Timeout: 10 * time.Second}, logger: slog.Default()}
}

func (s *ProviderSyncService) Sync(ctx context.Context, boletoID string) (*ProviderSyncResult, error) {
	if !IsValidUUID(boletoID) {
		return nil, ErrValidation
	}
	boleto, err := s.boletos.FindByID(boletoID)
	if err != nil {
		return nil, err
	}
	if boleto.ProviderID == nil || boleto.OurNumber == nil || strings.TrimSpace(*boleto.OurNumber) == "" {
		return nil, ErrValidation
	}
	adapter, err := s.adapter(boleto)
	if err != nil {
		return nil, err
	}
	summary, err := adapter.GetBoleto(ctx, providertypes.GetRequest{TenantID: boleto.TenantID, ProviderID: *boleto.ProviderID, OurNumber: *boleto.OurNumber})
	if err != nil {
		return nil, err
	}
	if err := s.boletos.MarkProviderSynced(boleto.ID); err != nil {
		return nil, err
	}
	updated := applyProviderSummary(boleto, summary)
	if !updated {
		return &ProviderSyncResult{Boleto: boleto}, nil
	}
	if err := s.boletos.Update(boleto); err != nil {
		return nil, err
	}
	notified, err := s.enqueueAndDeliver(ctx, boleto)
	if err != nil {
		s.logger.Error("tenant webhook delivery failed after provider sync", "boleto_id", boleto.ID, "error", err)
	}
	return &ProviderSyncResult{Boleto: boleto, Updated: true, TenantNotified: notified}, nil
}

func (s *ProviderSyncService) SyncPending(ctx context.Context, limit int) (int, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	var locked bool
	if err = conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(746492018221)`).Scan(&locked); err != nil {
		return 0, err
	}
	if !locked {
		return 0, nil
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(746492018221)`)
	items, err := s.boletos.ListForProviderSync(limit)
	if err != nil {
		return 0, err
	}
	updated := 0
	for i := range items {
		result, err := s.Sync(ctx, items[i].ID)
		if err != nil {
			s.logger.Error("periodic provider sync failed", "boleto_id", items[i].ID, "error", err)
			continue
		}
		if result.Updated {
			updated++
		}
	}
	_ = s.RetryPendingDeliveries(ctx, limit)
	return updated, nil
}

func (s *ProviderSyncService) adapter(b *domain.Boleto) (contracts.ProviderAdapter, error) {
	assignment, err := s.providers.FindTenantProvider(b.TenantID, *b.ProviderID)
	if err != nil {
		return nil, err
	}
	cfg := providertypes.ProviderConfig{ID: *b.ProviderID, TenantID: b.TenantID, Name: assignment.Provider.Name}
	if assignment.TenantProvider.Config != nil && strings.TrimSpace(*assignment.TenantProvider.Config) != "" {
		cfg.Config = strings.TrimSpace(*assignment.TenantProvider.Config)
	} else if assignment.Provider.Config != nil {
		cfg.Config = strings.TrimSpace(*assignment.Provider.Config)
	}
	return s.factory.Build(cfg)
}

func applyProviderSummary(b *domain.Boleto, summary providertypes.BoletoSummary) bool {
	changed := false
	set := func(target **string, value string) {
		value = strings.TrimSpace(value)
		if value != "" && (*target == nil || **target != value) {
			*target = &value
			changed = true
		}
	}
	set(&b.OurNumber, summary.OurNumber)
	set(&b.Barcode, summary.Barcode)
	set(&b.DigitableLine, summary.DigitableLine)
	set(&b.Base64, summary.Base64)
	if summary.Status != "" && string(summary.Status) != b.Status {
		b.Status = string(summary.Status)
		changed = true
	}
	if b.Status == string(providertypes.StatusIssued) && b.IssuedAt == nil {
		now := time.Now().UTC()
		b.IssuedAt = &now
		changed = true
	}
	return changed
}

func (s *ProviderSyncService) enqueueAndDeliver(ctx context.Context, b *domain.Boleto) (bool, error) {
	eventID := uuid.NewString()
	payload, _ := json.Marshal(map[string]any{"event_id": eventID, "event_type": "BOLETO_UPDATED", "occurred_at": time.Now().UTC(), "provider": "Moncalieri", "boleto": b})
	_, err := s.db.Exec(`INSERT INTO webhook_events(id,tenant_id,type,payload,provider_id,external_event_id,item_sequence) VALUES($1,$2,'BOLETO_PROVIDER_SYNC',$3,$4,$1,0)`, eventID, b.TenantID, string(payload), b.ProviderID)
	if err != nil {
		return false, err
	}
	if err = s.deliver(ctx, eventID, b.TenantID, payload); err != nil {
		s.markFailure(eventID, err)
		return false, err
	}
	return true, s.markDelivered(eventID)
}
func (s *ProviderSyncService) deliver(ctx context.Context, eventID, tenantID string, payload []byte) error {
	tenant, err := s.tenants.FindByID(tenantID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tenant.WebhookURL) == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tenant.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Giga-Event-ID", eventID)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("tenant webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}
func (s *ProviderSyncService) RetryPendingDeliveries(ctx context.Context, limit int) error {
	rows, err := s.db.Query(`SELECT external_event_id,tenant_id,payload FROM webhook_events WHERE type='BOLETO_PROVIDER_SYNC' AND delivered_at IS NULL ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return err
	}
	defer rows.Close()
	type pending struct{ id, tenant, payload string }
	items := []pending{}
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.tenant, &p.payload); err != nil {
			return err
		}
		items = append(items, p)
	}
	for _, p := range items {
		if err := s.deliver(ctx, p.id, p.tenant, []byte(p.payload)); err != nil {
			s.markFailure(p.id, err)
			continue
		}
		_ = s.markDelivered(p.id)
	}
	return rows.Err()
}
func (s *ProviderSyncService) markDelivered(id string) error {
	_, err := s.db.Exec(`UPDATE webhook_events SET delivered_at=now(),delivery_attempts=delivery_attempts+1,last_delivery_error=NULL WHERE external_event_id=$1 AND type='BOLETO_PROVIDER_SYNC'`, id)
	return err
}
func (s *ProviderSyncService) markFailure(id string, cause error) {
	_, _ = s.db.Exec(`UPDATE webhook_events SET delivery_attempts=delivery_attempts+1,last_delivery_error=$1 WHERE external_event_id=$2 AND type='BOLETO_PROVIDER_SYNC'`, cause.Error(), id)
}
