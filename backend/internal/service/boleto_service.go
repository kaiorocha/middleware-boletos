package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/base"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/contracts"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

type boletoRepo interface {
	Create(*domain.Boleto) error
	FindByID(string) (*domain.Boleto, error)
	ListByTenant(string) ([]domain.Boleto, error)
	Update(*domain.Boleto) error
	Delete(string, string) error
}

type blacklistCompliance interface {
	IsBlocked(string, string) (*domain.BlacklistEntry, bool, error)
	RecordBlockedEmissionAttempt(string, *domain.BlacklistEntry, *domain.Boleto)
}

type BoletoService struct {
	repo         boletoRepo
	customers    customerRepo
	providers    providerRepo
	blacklist    blacklistCompliance
	factory      contracts.ProviderFactory
	payerBuilder base.PayerBuilder
	logger       *slog.Logger
}

type adminBoletoReader interface {
	AdminDashboard(domain.BoletoFilters) (*domain.AdminDashboard, error)
	ListTransactions(domain.BoletoFilters) (*domain.PaginatedTransactions, error)
}

func NewBoletoService(repo boletoRepo) *BoletoService {
	return &BoletoService{repo: repo, payerBuilder: base.NewDefaultPayerBuilder(), logger: slog.Default()}
}

func (s *BoletoService) WithCustomerRepository(repo customerRepo) *BoletoService {
	s.customers = repo
	return s
}

func (s *BoletoService) WithProviderRepository(repo providerRepo) *BoletoService {
	s.providers = repo
	return s
}

func (s *BoletoService) WithBlacklistService(blacklist blacklistCompliance) *BoletoService {
	s.blacklist = blacklist
	return s
}

func (s *BoletoService) WithProviderFactory(factory contracts.ProviderFactory) *BoletoService {
	s.factory = factory
	return s
}

func (s *BoletoService) WithPayerBuilder(builder base.PayerBuilder) *BoletoService {
	if builder != nil {
		s.payerBuilder = builder
	}
	return s
}

func (s *BoletoService) WithLogger(logger *slog.Logger) *BoletoService {
	if logger != nil {
		s.logger = logger
	}
	return s
}

func (s *BoletoService) Create(b *domain.Boleto) error {
	b.ExternalID = NormalizeOptionalString(b.ExternalID)
	b.OurNumber = NormalizeOptionalString(b.OurNumber)
	if !IsValidUUID(b.TenantID) || !IsValidUUID(b.CustomerID) {
		return ErrValidation
	}
	if b.ProviderID != nil && !IsValidUUID(*b.ProviderID) {
		return ErrValidation
	}
	if b.AmountCents <= 0 {
		return ErrValidation
	}
	if b.DueDate.IsZero() {
		return ErrValidation
	}

	b.Status = strings.ToUpper(strings.TrimSpace(b.Status))
	if b.Status == "" {
		b.Status = "CREATED"
	}
	if !base.IsKnownStatus(types.BoletoStatus(b.Status)) {
		return ErrValidation
	}

	return s.repo.Create(b)
}

func (s *BoletoService) Get(id string) (*domain.Boleto, error) {
	if !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.FindByID(id)
}

func (s *BoletoService) ListByTenant(tenantID string) ([]domain.Boleto, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	return s.repo.ListByTenant(tenantID)
}

func (s *BoletoService) AdminDashboard(filters domain.BoletoFilters) (*domain.AdminDashboard, error) {
	if err := validateBoletoFilters(filters); err != nil {
		return nil, err
	}
	reader, ok := s.repo.(adminBoletoReader)
	if !ok {
		return nil, ErrValidation
	}
	return reader.AdminDashboard(filters)
}

func (s *BoletoService) ListTransactions(filters domain.BoletoFilters) (*domain.PaginatedTransactions, error) {
	if err := validateBoletoFilters(filters); err != nil {
		return nil, err
	}
	reader, ok := s.repo.(adminBoletoReader)
	if !ok {
		return nil, ErrValidation
	}
	return reader.ListTransactions(filters)
}

func (s *BoletoService) Emit(ctx context.Context, tenantID, boletoID string) (*domain.Boleto, error) {
	if !IsValidUUID(tenantID) || !IsValidUUID(boletoID) {
		return nil, ErrValidation
	}
	if s.customers == nil || s.providers == nil || s.blacklist == nil || s.factory == nil || s.payerBuilder == nil {
		return nil, ErrValidation
	}

	start := time.Now()
	boleto, err := s.repo.FindByID(boletoID)
	if err != nil {
		return nil, err
	}
	if boleto.TenantID != tenantID || boleto.ProviderID == nil || !IsValidUUID(*boleto.ProviderID) {
		return nil, ErrValidation
	}

	if boleto.Status == string(types.StatusIssued) && boleto.ExternalID != nil && boleto.OurNumber != nil {
		s.logger.Info("boleto emission idempotent hit",
			"tenant", tenantID,
			"provider", *boleto.ProviderID,
			"request_id", requestID(ctx),
			"boleto_id", boleto.ID,
			"latency_ms", time.Since(start).Milliseconds(),
			"result", "already_issued",
		)
		return boleto, nil
	}

	if !base.CanTransition(types.BoletoStatus(boleto.Status), types.StatusProcessing) {
		return nil, ErrValidation
	}

	customer, err := s.customers.FindByID(boleto.CustomerID)
	if err != nil {
		return nil, err
	}
	if customer.TenantID != tenantID {
		return nil, ErrValidation
	}

	if customer.Document == nil {
		return nil, ErrValidation
	}
	document := normalizeDocumentValue(*customer.Document)
	if document == "" {
		return nil, ErrValidation
	}
	entry, blocked, err := s.blacklist.IsBlocked(tenantID, document)
	if err != nil {
		return nil, err
	}
	if blocked {
		s.blacklist.RecordBlockedEmissionAttempt(tenantID, entry, boleto)
		s.logger.Info("boleto emission blocked by compliance",
			"tenant", tenantID,
			"request_id", requestID(ctx),
			"boleto_id", boleto.ID,
			"customer_id", customer.ID,
			"latency_ms", time.Since(start).Milliseconds(),
			"result", "blocked",
		)
		return nil, NewCustomerBlocked("Este cliente está bloqueado para novas emissões.")
	}

	payer, err := s.payerBuilder.Build(*customer)
	if err != nil {
		return nil, err
	}

	provider, err := s.providers.FindByID(*boleto.ProviderID)
	if err != nil {
		return nil, err
	}
	if provider.Status != "ACTIVE" {
		return nil, ErrProviderNotAllowed
	}
	allowed, err := s.providers.IsAllowedForTenant(tenantID, *boleto.ProviderID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrProviderNotAllowed
	}

	cfg := types.ProviderConfig{ID: provider.ID, TenantID: tenantID, Name: provider.Name}
	if provider.Config != nil {
		cfg.Config = *provider.Config
	}
	adapter, err := s.factory.Build(cfg)
	if err != nil {
		return nil, err
	}

	boleto.Status = string(types.StatusProcessing)
	if err := s.repo.Update(boleto); err != nil {
		return nil, err
	}

	response, err := adapter.IssueBoleto(ctx, types.IssueRequest{
		TenantID:    boleto.TenantID,
		BoletoID:    boleto.ID,
		CustomerID:  boleto.CustomerID,
		ExternalID:  optionalStringValue(boleto.ExternalID),
		AmountCents: boleto.AmountCents,
		DueDate:     boleto.DueDate,
		Payer:       payer,
	})
	if err != nil {
		boleto.Status = string(types.StatusFailed)
		_ = s.repo.Update(boleto)
		s.logger.Error("boleto emission failed",
			"tenant", tenantID,
			"provider", provider.Name,
			"request_id", requestID(ctx),
			"boleto_id", boleto.ID,
			"latency_ms", time.Since(start).Milliseconds(),
			"result", "failed",
			"error", err.Error(),
		)
		return nil, err
	}
	if !base.CanTransition(types.StatusProcessing, response.Status) {
		return nil, ErrValidation
	}

	boleto.Status = string(response.Status)
	boleto.ExternalID = stringPtr(response.ExternalID)
	boleto.Barcode = stringPtr(response.Barcode)
	boleto.DigitableLine = stringPtr(response.DigitableLine)
	boleto.OurNumber = stringPtr(response.OurNumber)
	if !response.IssuedAt.IsZero() {
		boleto.IssuedAt = &response.IssuedAt
	}
	if err := s.repo.Update(boleto); err != nil {
		return nil, err
	}

	s.logger.Info("boleto emission completed",
		"tenant", tenantID,
		"provider", provider.Name,
		"request_id", requestID(ctx),
		"boleto_id", boleto.ID,
		"latency_ms", time.Since(start).Milliseconds(),
		"result", boleto.Status,
	)
	return boleto, nil
}

func validateBoletoFilters(filters domain.BoletoFilters) error {
	if filters.TenantID != "" && !IsValidUUID(filters.TenantID) {
		return ErrValidation
	}
	if filters.ProviderID != "" && !IsValidUUID(filters.ProviderID) {
		return ErrValidation
	}
	if filters.From != nil && filters.To != nil && filters.From.After(*filters.To) {
		return ErrValidation
	}
	return nil
}

func NormalizeDueDate(dateOnly string) (time.Time, error) {
	return time.Parse("2006-01-02", dateOnly)
}

type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func requestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

func stringPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v := s
	return &v
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
