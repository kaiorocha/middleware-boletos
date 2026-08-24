package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/contracts"
	providertypes "github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

type moncalieriBoletoRepo interface {
	FindByProviderReference(string, string, string) (*domain.Boleto, error)
	Update(*domain.Boleto) error
}
type moncalieriTenantRepo interface {
	FindByID(string) (*domain.Tenant, error)
}
type moncalieriProviderRepo interface {
	FindTenantProvider(string, string) (*domain.TenantProviderConfig, error)
}

type MoncalieriWebhookService struct {
	db         *sql.DB
	boletos    moncalieriBoletoRepo
	tenants    moncalieriTenantRepo
	providers  moncalieriProviderRepo
	factory    contracts.ProviderFactory
	httpClient *http.Client
}

func NewMoncalieriWebhookService(db *sql.DB, boletos moncalieriBoletoRepo, tenants moncalieriTenantRepo, providers moncalieriProviderRepo, factory contracts.ProviderFactory) *MoncalieriWebhookService {
	return &MoncalieriWebhookService{db: db, boletos: boletos, tenants: tenants, providers: providers, factory: factory, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

type moncalieriRegistration struct {
	EventID    string                       `json:"eventId"`
	EventType  string                       `json:"eventType"`
	OccurredAt time.Time                    `json:"occurredAt"`
	Integrated []moncalieriRegistrationItem `json:"integrados"`
	Errors     []moncalieriRegistrationItem `json:"erros"`
}
type moncalieriRegistrationItem struct {
	Sequence          int       `json:"sequencia"`
	CustomerReference string    `json:"seuNumero"`
	OurNumber         string    `json:"nossoNumero"`
	Barcode           string    `json:"codigoBarras"`
	DigitableLine     string    `json:"linhaDigitavel"`
	Base64            string    `json:"base64"`
	BoletoBase64      string    `json:"boletoBase64"`
	PDFBase64         string    `json:"pdfBase64"`
	FileBase64        string    `json:"arquivoBase64"`
	EventDate         time.Time `json:"dataEvento"`
	Error             string    `json:"erro"`
}

func (i moncalieriRegistrationItem) boletoBase64() string {
	for _, value := range []string{i.Base64, i.BoletoBase64, i.PDFBase64, i.FileBase64} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func (s *MoncalieriWebhookService) Receive(ctx context.Context, providerID string, body []byte) error {
	if _, err := uuid.Parse(providerID); err != nil {
		return ErrValidation
	}
	var event moncalieriRegistration
	if json.Unmarshal(body, &event) != nil {
		return ErrValidation
	}
	if _, err := uuid.Parse(event.EventID); err != nil || !strings.EqualFold(event.EventType, "REGISTRO") || len(event.Integrated)+len(event.Errors) == 0 {
		return ErrValidation
	}
	for _, item := range event.Integrated {
		if err := s.processItem(ctx, providerID, event, item, false, body); err != nil {
			return err
		}
	}
	for _, item := range event.Errors {
		if err := s.processItem(ctx, providerID, event, item, true, body); err != nil {
			return err
		}
	}
	return nil
}

func (s *MoncalieriWebhookService) processItem(ctx context.Context, providerID string, event moncalieriRegistration, item moncalieriRegistrationItem, failed bool, raw []byte) error {
	boleto, err := s.boletos.FindByProviderReference(providerID, strings.TrimSpace(item.CustomerReference), strings.TrimSpace(item.OurNumber))
	if err != nil {
		return err
	}
	if !failed && (strings.TrimSpace(item.DigitableLine) == "" || item.boletoBase64() == "") {
		if err := s.completeFromProvider(ctx, providerID, boleto, &item); err != nil {
			return err
		}
	}
	inserted, delivered, err := s.reserveEvent(providerID, event.EventID, item.Sequence, boleto.TenantID, event.EventType, raw)
	if err != nil {
		return err
	}
	if inserted {
		if failed {
			boleto.Status = string(providertypes.StatusFailed)
		} else {
			boleto.Status = string(providertypes.StatusIssued)
			setWebhookValue(&boleto.OurNumber, item.OurNumber)
			setWebhookValue(&boleto.Barcode, item.Barcode)
			setWebhookValue(&boleto.DigitableLine, item.DigitableLine)
			setWebhookValue(&boleto.Base64, item.boletoBase64())
			issuedAt := item.EventDate
			if issuedAt.IsZero() {
				issuedAt = event.OccurredAt
			}
			if issuedAt.IsZero() {
				issuedAt = time.Now().UTC()
			}
			boleto.IssuedAt = &issuedAt
		}
		if err := s.boletos.Update(boleto); err != nil {
			s.releaseEvent(providerID, event.EventID, item.Sequence)
			return err
		}
	}
	if delivered {
		return nil
	}
	tenant, err := s.tenants.FindByID(boleto.TenantID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tenant.WebhookURL) == "" {
		return s.markDelivered(providerID, event.EventID, item.Sequence)
	}
	payload := map[string]any{"event_id": event.EventID, "event_type": "BOLETO_REGISTRATION", "occurred_at": event.OccurredAt, "provider": "Moncalieri", "boleto": boleto}
	if failed {
		payload["event_type"] = "BOLETO_REGISTRATION_FAILED"
		payload["error"] = item.Error
	}
	encoded, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tenant.WebhookURL, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Giga-Event-ID", event.EventID)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.markDeliveryFailure(providerID, event.EventID, item.Sequence, err)
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("tenant webhook returned HTTP %d", resp.StatusCode)
		s.markDeliveryFailure(providerID, event.EventID, item.Sequence, err)
		return err
	}
	return s.markDelivered(providerID, event.EventID, item.Sequence)
}

func (s *MoncalieriWebhookService) completeFromProvider(ctx context.Context, providerID string, boleto *domain.Boleto, item *moncalieriRegistrationItem) error {
	if s.providers == nil || s.factory == nil {
		return fmt.Errorf("Moncalieri consultation is not configured")
	}
	assignment, err := s.providers.FindTenantProvider(boleto.TenantID, providerID)
	if err != nil {
		return err
	}
	config := providertypes.ProviderConfig{ID: providerID, TenantID: boleto.TenantID, Name: assignment.Provider.Name}
	if assignment.TenantProvider.Config != nil && strings.TrimSpace(*assignment.TenantProvider.Config) != "" {
		config.Config = strings.TrimSpace(*assignment.TenantProvider.Config)
	} else if assignment.Provider.Config != nil {
		config.Config = strings.TrimSpace(*assignment.Provider.Config)
	}
	adapter, err := s.factory.Build(config)
	if err != nil {
		return err
	}
	summary, err := adapter.GetBoleto(ctx, providertypes.GetRequest{TenantID: boleto.TenantID, ProviderID: providerID, OurNumber: item.OurNumber})
	if err != nil {
		return err
	}
	if item.Barcode == "" {
		item.Barcode = summary.Barcode
	}
	if item.DigitableLine == "" {
		item.DigitableLine = summary.DigitableLine
	}
	if item.boletoBase64() == "" {
		item.Base64 = summary.Base64
	}
	if strings.TrimSpace(item.DigitableLine) == "" || item.boletoBase64() == "" {
		return fmt.Errorf("registered boleto consultation did not return digitable line and base64")
	}
	return nil
}
func setWebhookValue(target **string, value string) {
	if value = strings.TrimSpace(value); value != "" {
		*target = &value
	}
}
func (s *MoncalieriWebhookService) reserveEvent(providerID, eventID string, sequence int, tenantID, eventType string, raw []byte) (bool, bool, error) {
	result, err := s.db.Exec(`INSERT INTO webhook_events(id,tenant_id,type,payload,provider_id,external_event_id,item_sequence) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(provider_id,external_event_id,item_sequence) WHERE provider_id IS NOT NULL AND external_event_id IS NOT NULL DO NOTHING`, uuid.NewString(), tenantID, eventType, string(raw), providerID, eventID, sequence)
	if err != nil {
		return false, false, err
	}
	affected, _ := result.RowsAffected()
	var delivered sql.NullTime
	err = s.db.QueryRow(`SELECT delivered_at FROM webhook_events WHERE provider_id=$1 AND external_event_id=$2 AND item_sequence=$3`, providerID, eventID, sequence).Scan(&delivered)
	return affected == 1, delivered.Valid, err
}
func (s *MoncalieriWebhookService) markDelivered(providerID, eventID string, sequence int) error {
	_, err := s.db.Exec(`UPDATE webhook_events SET delivered_at=now(),delivery_attempts=delivery_attempts+1,last_delivery_error=NULL WHERE provider_id=$1 AND external_event_id=$2 AND item_sequence=$3`, providerID, eventID, sequence)
	return err
}
func (s *MoncalieriWebhookService) markDeliveryFailure(providerID, eventID string, sequence int, cause error) {
	_, _ = s.db.Exec(`UPDATE webhook_events SET delivery_attempts=delivery_attempts+1,last_delivery_error=$1 WHERE provider_id=$2 AND external_event_id=$3 AND item_sequence=$4`, cause.Error(), providerID, eventID, sequence)
}

func (s *MoncalieriWebhookService) releaseEvent(providerID, eventID string, sequence int) {
	_, _ = s.db.Exec(`DELETE FROM webhook_events WHERE provider_id=$1 AND external_event_id=$2 AND item_sequence=$3 AND delivery_attempts=0`, providerID, eventID, sequence)
}
