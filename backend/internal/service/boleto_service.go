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
	IsBlockedByDocument(string, string) (*domain.BlacklistEntry, bool, error)
	IsBlockedByEmail(string, string) (*domain.BlacklistEntry, bool, error)
	RecordBlockedEmissionAttempt(string, *domain.BlacklistEntry, *domain.Boleto)
}

type BoletoService struct {
	repo      boletoRepo
	customers customerRepo
	providers providerRepo
	tenants   interface {
		FindByID(string) (*domain.Tenant, error)
	}
	blacklist    blacklistCompliance
	factory      contracts.ProviderFactory
	payerBuilder base.PayerBuilder
	logger       *slog.Logger
}

type adminBoletoReader interface {
	AdminDashboard(domain.BoletoFilters) (*domain.AdminDashboard, error)
	ListTransactions(domain.BoletoFilters) (*domain.PaginatedTransactions, error)
}

type tenantProviderReader interface {
	FindTenantProvider(string, string) (*domain.TenantProviderConfig, error)
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

func (s *BoletoService) WithTenantRepository(repo interface {
	FindByID(string) (*domain.Tenant, error)
}) *BoletoService {
	s.tenants = repo
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

	if !IsValidUUID(b.TenantID) {
		return ErrValidation
	}

	// CustomerID is now optional (for proposal boletos)
	// Either CustomerID or RecipientEmail must be provided
	if b.CustomerID != nil && !IsValidUUID(*b.CustomerID) {
		return ErrValidation
	}

	// Normalize and validate RecipientEmail
	b.RecipientEmail = NormalizeEmail(b.RecipientEmail)
	if b.RecipientEmail == "" && b.CustomerID == nil {
		return ErrValidation
	}
	if b.RecipientEmail != "" && !IsValidEmail(b.RecipientEmail) {
		return ErrValidation
	}

	// If only RecipientEmail is provided (no CustomerID), it must be valid
	// If both are provided, validate both
	if b.CustomerID == nil && b.RecipientEmail == "" {
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
	if s.providers == nil || s.blacklist == nil || s.factory == nil {
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

	var fallbackPayer *types.Payer

	// Case A: Traditional boleto with customer
	if boleto.CustomerID != nil {
		if s.customers == nil || s.payerBuilder == nil {
			return nil, ErrValidation
		}
		if !IsValidUUID(*boleto.CustomerID) {
			return nil, ErrValidation
		}

		customer, err := s.customers.FindByID(*boleto.CustomerID)
		if err != nil {
			return nil, err
		}
		if customer.TenantID != tenantID {
			return nil, ErrValidation
		}

		// Validate document exists
		if customer.Document == nil {
			return nil, ErrValidation
		}
		document := normalizeDocumentValue(*customer.Document)
		if document == "" {
			return nil, ErrValidation
		}

		// Check compliance - blocked by document
		entry, blocked, err := s.blacklist.IsBlockedByDocument(tenantID, document)
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
		fallbackPayer, err = s.payerBuilder.Build(*customer)
		if err != nil {
			return nil, err
		}

	} else if boleto.RecipientEmail != "" {
		// Case B: Proposal boleto with recipient email only
		email := NormalizeEmail(boleto.RecipientEmail)
		if !IsValidEmail(email) {
			return nil, ErrValidation
		}

		// Check compliance - blocked by email
		entry, blocked, err := s.blacklist.IsBlockedByEmail(tenantID, email)
		if err != nil {
			return nil, err
		}
		if blocked {
			s.blacklist.RecordBlockedEmissionAttempt(tenantID, entry, boleto)
			s.logger.Info("boleto emission blocked by compliance (recipient)",
				"tenant", tenantID,
				"request_id", requestID(ctx),
				"boleto_id", boleto.ID,
				"recipient_email", email,
				"latency_ms", time.Since(start).Milliseconds(),
				"result", "blocked",
			)
			return nil, NewRecipientBlocked("Este destinatário está bloqueado para novas emissões.")
		}
		fallbackPayer = &types.Payer{Email: email}

	} else {
		// Neither CustomerID nor RecipientEmail provided
		return nil, ErrValidation
	}

	payer := fallbackPayer
	if s.tenants != nil {
		tenant, err := s.tenants.FindByID(tenantID)
		if err != nil || tenant.ID != "" && tenant.ID != tenantID {
			return nil, ErrValidation
		}
		payer = &types.Payer{
			Document: tenant.Document, Name: tenant.Name, Address: tenant.Address,
			District: tenant.District, City: tenant.City, PostalCode: tenant.PostalCode,
			State: tenant.State, CountryCode: tenant.CountryCode, AreaCode: tenant.AreaCode,
			PhoneNumber: tenant.PhoneNumber, Email: NormalizeEmail(boleto.RecipientEmail),
		}
	}

	providerConfig, err := s.providerConfigForTenant(tenantID, *boleto.ProviderID)
	if err != nil {
		return nil, err
	}
	adapter, err := s.factory.Build(providerConfig)
	if err != nil {
		return nil, err
	}

	boleto.Status = string(types.StatusProcessing)
	if err := s.repo.Update(boleto); err != nil {
		return nil, err
	}

	response, err := adapter.IssueBoleto(ctx, types.IssueRequest{
		TenantID:       boleto.TenantID,
		BoletoID:       boleto.ID,
		CustomerID:     optionalStringValue(boleto.CustomerID),
		RecipientEmail: boleto.RecipientEmail,
		ExternalID:     optionalStringValue(boleto.ExternalID),
		AmountCents:    boleto.AmountCents,
		DueDate:        boleto.DueDate,
		Payer:          payer,
	})
	if err != nil {
		boleto.Status = string(types.StatusFailed)
		_ = s.repo.Update(boleto)
		s.logger.Error("boleto emission failed",
			"tenant", tenantID,
			"provider", providerConfig.Name,
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
	boleto.Base64 = stringPtr(response.Base64)
	if !response.IssuedAt.IsZero() {
		boleto.IssuedAt = &response.IssuedAt
	}
	if err := s.repo.Update(boleto); err != nil {
		return nil, err
	}

	s.logger.Info("boleto emission completed",
		"tenant", tenantID,
		"provider", providerConfig.Name,
		"request_id", requestID(ctx),
		"boleto_id", boleto.ID,
		"latency_ms", time.Since(start).Milliseconds(),
		"result", boleto.Status,
	)
	return boleto, nil
}

func (s *BoletoService) providerConfigForTenant(tenantID, providerID string) (types.ProviderConfig, error) {
	cfg := types.ProviderConfig{ID: providerID, TenantID: tenantID}
	reader, ok := s.providers.(tenantProviderReader)
	if ok {
		tenantProviderConfig, err := reader.FindTenantProvider(tenantID, providerID)
		if err != nil {
			return cfg, ErrProviderNotAllowed
		}
		provider := tenantProviderConfig.Provider
		assignment := tenantProviderConfig.TenantProvider
		if provider.Status != "ACTIVE" || assignment.DeletedAt != nil || !assignment.Active {
			return cfg, ErrProviderNotAllowed
		}
		cfg.Name = provider.Name
		if assignment.Config != nil && strings.TrimSpace(*assignment.Config) != "" {
			cfg.Config = strings.TrimSpace(*assignment.Config)
		} else if provider.Config != nil && strings.TrimSpace(*provider.Config) != "" {
			cfg.Config = strings.TrimSpace(*provider.Config)
		}
		return cfg, nil
	}

	provider, err := s.providers.FindByID(providerID)
	if err != nil {
		return cfg, err
	}
	if provider.Status != "ACTIVE" {
		return cfg, ErrProviderNotAllowed
	}
	allowed, err := s.providers.IsAllowedForTenant(tenantID, providerID)
	if err != nil {
		return cfg, err
	}
	if !allowed {
		return cfg, ErrProviderNotAllowed
	}
	cfg.Name = provider.Name
	if provider.Config != nil {
		cfg.Config = strings.TrimSpace(*provider.Config)
	}
	return cfg, nil
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
