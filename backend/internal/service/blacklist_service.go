package service

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type blacklistRepo interface {
	Create(*domain.BlacklistEntry) error
	FindByID(string, string) (*domain.BlacklistEntry, error)
	FindByDocument(string, string) (*domain.BlacklistEntry, error)
	FindByType(string, string, string) (*domain.BlacklistEntry, error)
	List(string, string, *bool) ([]domain.BlacklistEntry, error)
	Update(*domain.BlacklistEntry) error
	SoftDelete(string, string) error
	IsBlocked(string, string) (*domain.BlacklistEntry, bool, error)
	IsBlockedByType(string, string, string) (*domain.BlacklistEntry, bool, error)
}

type auditRepo interface {
	Create(*domain.AuditLog) error
}

type BlacklistService struct {
	repo  blacklistRepo
	audit auditRepo
}

func NewBlacklistService(repo blacklistRepo) *BlacklistService {
	return &BlacklistService{repo: repo}
}

func (s *BlacklistService) WithAuditRepository(repo auditRepo) *BlacklistService {
	s.audit = repo
	return s
}

func (s *BlacklistService) Create(entry *domain.BlacklistEntry) error {
	if err := normalizeBlacklistEntry(entry); err != nil {
		return err
	}
	if !entry.Active {
		entry.Active = true
	}
	if err := s.repo.Create(entry); err != nil {
		return err
	}
	action := auditActionForBlock(entry.EntryType)
	metadata := blacklistAuditMetadata(entry)
	s.auditCompliance(entry.TenantID, entry.CreatedBy, action, metadata)
	return nil
}

func (s *BlacklistService) Get(tenantID, id string) (*domain.BlacklistEntry, error) {
	if !IsValidUUID(tenantID) || !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.FindByID(tenantID, id)
}

func (s *BlacklistService) GetByDocument(tenantID, document string) (*domain.BlacklistEntry, error) {
	document = normalizeDocumentValue(document)
	if !IsValidUUID(tenantID) || document == "" {
		return nil, ErrValidation
	}
	return s.repo.FindByDocument(tenantID, document)
}

func (s *BlacklistService) List(tenantID, search string, active *bool) ([]domain.BlacklistEntry, error) {
	if !IsValidUUID(tenantID) {
		return nil, ErrValidation
	}
	search = strings.TrimSpace(search)
	return s.repo.List(tenantID, search, active)
}

func (s *BlacklistService) Update(entry *domain.BlacklistEntry) error {
	if entry.ID == "" {
		return ErrValidation
	}
	if err := normalizeBlacklistEntry(entry); err != nil {
		return err
	}
	return s.repo.Update(entry)
}

func (s *BlacklistService) Delete(tenantID, id string) error {
	if !IsValidUUID(tenantID) || !IsValidUUID(id) {
		return ErrValidation
	}
	return s.repo.SoftDelete(tenantID, id)
}

func (s *BlacklistService) Block(tenantID, id string, createdBy *string) (*domain.BlacklistEntry, error) {
	entry, err := s.Get(tenantID, id)
	if err != nil {
		return nil, err
	}
	entry.Active = true
	if createdBy != nil {
		entry.CreatedBy = createdBy
	}
	if err := s.repo.Update(entry); err != nil {
		return nil, err
	}
	action := auditActionForBlock(entry.EntryType)
	metadata := blacklistAuditMetadata(entry)
	s.auditCompliance(tenantID, createdBy, action, metadata)
	return entry, nil
}

func (s *BlacklistService) Unblock(tenantID, id string, createdBy *string) (*domain.BlacklistEntry, error) {
	entry, err := s.Get(tenantID, id)
	if err != nil {
		return nil, err
	}
	entry.Active = false
	if createdBy != nil {
		entry.CreatedBy = createdBy
	}
	if err := s.repo.Update(entry); err != nil {
		return nil, err
	}
	action := auditActionForUnblock(entry.EntryType)
	metadata := blacklistAuditMetadata(entry)
	s.auditCompliance(tenantID, createdBy, action, metadata)
	return entry, nil
}

func (s *BlacklistService) IsBlocked(tenantID, document string) (*domain.BlacklistEntry, bool, error) {
	document = normalizeDocumentValue(document)
	if !IsValidUUID(tenantID) || document == "" {
		return nil, false, ErrValidation
	}
	return s.repo.IsBlocked(tenantID, document)
}

func (s *BlacklistService) IsBlockedByDocument(tenantID, document string) (*domain.BlacklistEntry, bool, error) {
	document = normalizeDocumentValue(document)
	if !IsValidUUID(tenantID) || document == "" {
		return nil, false, ErrValidation
	}
	return s.repo.IsBlocked(tenantID, document)
}

func (s *BlacklistService) IsBlockedByEmail(tenantID, email string) (*domain.BlacklistEntry, bool, error) {
	email = NormalizeEmail(email)
	if !IsValidUUID(tenantID) || email == "" {
		return nil, false, ErrValidation
	}
	if !IsValidEmail(email) {
		return nil, false, ErrValidation
	}
	return s.repo.IsBlockedByType(tenantID, "EMAIL", email)
}

func (s *BlacklistService) RecordBlockedEmissionAttempt(tenantID string, entry *domain.BlacklistEntry, boleto *domain.Boleto) {
	if entry == nil || boleto == nil {
		return
	}
	metadata := map[string]any{
		"entry_type":  entry.EntryType,
		"boleto_id":   boleto.ID,
		"provider_id": boleto.ProviderID,
		"reason":      entry.Reason,
	}

	// Add blocked_value (prefer ValueNormalized, fallback to Value, then Document)
	blockedValue := entry.ValueNormalized
	if blockedValue == "" {
		blockedValue = entry.Value
	}
	if blockedValue == "" && entry.EntryType == "DOCUMENT" {
		blockedValue = entry.Document
	}
	if blockedValue != "" {
		metadata["blocked_value"] = blockedValue
	}

	// Add recipient_email if present (for EMAIL blocks)
	if boleto.RecipientEmail != "" {
		metadata["recipient_email"] = boleto.RecipientEmail
	}

	// Add customer_id if present
	if boleto.CustomerID != nil && *boleto.CustomerID != "" {
		metadata["customer_id"] = *boleto.CustomerID
	}

	// Add document if DOCUMENT type
	if entry.EntryType == "DOCUMENT" && entry.Document != "" {
		metadata["document"] = entry.Document
	}

	s.auditCompliance(tenantID, nil, "BlockedEmissionAttempt", metadata)
}

func normalizeBlacklistEntry(entry *domain.BlacklistEntry) error {
	if entry == nil || !IsValidUUID(entry.TenantID) {
		return ErrValidation
	}

	// Validate entry_type
	entryType := strings.ToUpper(strings.TrimSpace(entry.EntryType))
	if entryType == "" {
		// Legacy support: if Document is provided, default to DOCUMENT type
		if entry.Document != "" {
			entryType = "DOCUMENT"
		} else if entry.Value != "" {
			// Try to infer from Value
			if IsValidEmail(entry.Value) {
				entryType = "EMAIL"
			} else {
				entryType = "DOCUMENT"
			}
		} else {
			return ErrValidation
		}
	}

	if entryType != "DOCUMENT" && entryType != "EMAIL" {
		return ErrValidation
	}
	entry.EntryType = entryType

	// Normalize based on type
	if entryType == "DOCUMENT" {
		doc := normalizeDocumentValue(entry.Document)
		if entry.Value != "" && entry.Value != entry.Document {
			// If Value is different, use it, but still normalize as document
			doc = normalizeDocumentValue(entry.Value)
		}
		if doc == "" {
			return ErrValidation
		}
		entry.Value = entry.Document // Keep original for display
		if entry.Value == "" {
			entry.Value = doc
		}
		entry.ValueNormalized = doc
	} else if entryType == "EMAIL" {
		email := NormalizeEmail(entry.Value)
		if email == "" {
			return ErrValidation
		}
		if !IsValidEmail(email) {
			return ErrValidation
		}
		entry.Value = email
		entry.ValueNormalized = email
	}

	entry.Name = strings.TrimSpace(entry.Name)
	entry.Reason = strings.TrimSpace(entry.Reason)
	entry.Notes = NormalizeOptionalString(entry.Notes)
	entry.Source = strings.ToUpper(strings.TrimSpace(entry.Source))
	if entry.Source == "" {
		entry.Source = "MANUAL"
	}
	if !isValidBlacklistSource(entry.Source) {
		return ErrValidation
	}
	if entry.CreatedBy != nil {
		v := strings.TrimSpace(*entry.CreatedBy)
		if v == "" {
			entry.CreatedBy = nil
		} else {
			entry.CreatedBy = &v
		}
	}
	return nil
}

func normalizeDocumentValue(document string) string {
	normalized := NormalizeDocument(&document)
	if normalized == nil {
		return ""
	}
	return *normalized
}

func isValidBlacklistSource(source string) bool {
	switch source {
	case "MANUAL", "API", "IMPORT":
		return true
	default:
		return false
	}
}

func auditActionForBlock(entryType string) string {
	if strings.ToUpper(entryType) == "EMAIL" {
		return "RecipientBlocked"
	}
	return "CustomerBlocked"
}

func auditActionForUnblock(entryType string) string {
	if strings.ToUpper(entryType) == "EMAIL" {
		return "RecipientUnblocked"
	}
	return "CustomerUnblocked"
}

func blacklistAuditMetadata(entry *domain.BlacklistEntry) map[string]any {
	if entry == nil {
		return map[string]any{}
	}
	metadata := map[string]any{
		"entry_type":       entry.EntryType,
		"value":            entry.Value,
		"value_normalized": entry.ValueNormalized,
		"reason":           entry.Reason,
		"source":           entry.Source,
	}

	// Add name if present
	if entry.Name != "" {
		metadata["name"] = entry.Name
	}

	// For DOCUMENT type, also include document field for backward compatibility
	if strings.ToUpper(entry.EntryType) == "DOCUMENT" && entry.Document != "" {
		metadata["document"] = entry.Document
	}

	return metadata
}

func (s *BlacklistService) auditCompliance(tenantID string, userID *string, action string, metadata map[string]any) {
	if s.audit == nil || !IsValidUUID(tenantID) {
		return
	}
	payload, err := json.Marshal(sortedMap(metadata))
	if err != nil {
		return
	}
	meta := string(payload)
	_ = s.audit.Create(&domain.AuditLog{
		TenantID: tenantID,
		UserID:   userID,
		Action:   action,
		Metadata: &meta,
	})
}

func sortedMap(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]any, len(metadata))
	for _, key := range keys {
		out[key] = metadata[key]
	}
	return out
}
