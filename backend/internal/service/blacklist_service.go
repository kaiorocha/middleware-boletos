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
	List(string, string, *bool) ([]domain.BlacklistEntry, error)
	Update(*domain.BlacklistEntry) error
	SoftDelete(string, string) error
	IsBlocked(string, string) (*domain.BlacklistEntry, bool, error)
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
	s.auditCompliance(entry.TenantID, entry.CreatedBy, "CustomerBlocked", map[string]any{
		"document": entry.Document,
		"name":     entry.Name,
		"reason":   entry.Reason,
		"source":   entry.Source,
	})
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
	s.auditCompliance(tenantID, createdBy, "CustomerBlocked", map[string]any{
		"document": entry.Document,
		"name":     entry.Name,
		"reason":   entry.Reason,
		"source":   entry.Source,
	})
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
	s.auditCompliance(tenantID, createdBy, "CustomerUnblocked", map[string]any{
		"document": entry.Document,
		"name":     entry.Name,
		"reason":   entry.Reason,
		"source":   entry.Source,
	})
	return entry, nil
}

func (s *BlacklistService) IsBlocked(tenantID, document string) (*domain.BlacklistEntry, bool, error) {
	document = normalizeDocumentValue(document)
	if !IsValidUUID(tenantID) || document == "" {
		return nil, false, ErrValidation
	}
	return s.repo.IsBlocked(tenantID, document)
}

func (s *BlacklistService) RecordBlockedEmissionAttempt(tenantID string, entry *domain.BlacklistEntry, boleto *domain.Boleto) {
	if entry == nil || boleto == nil {
		return
	}
	metadata := map[string]any{
		"customer_id": boleto.CustomerID,
		"document":    entry.Document,
		"boleto_id":   boleto.ID,
		"provider_id": boleto.ProviderID,
		"reason":      entry.Reason,
	}
	s.auditCompliance(tenantID, nil, "BlockedEmissionAttempt", metadata)
}

func normalizeBlacklistEntry(entry *domain.BlacklistEntry) error {
	if entry == nil || !IsValidUUID(entry.TenantID) {
		return ErrValidation
	}
	entry.Document = normalizeDocumentValue(entry.Document)
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Reason = strings.TrimSpace(entry.Reason)
	entry.Notes = NormalizeOptionalString(entry.Notes)
	entry.Source = strings.ToUpper(strings.TrimSpace(entry.Source))
	if entry.Source == "" {
		entry.Source = "MANUAL"
	}
	if !isValidBlacklistSource(entry.Source) || entry.Document == "" {
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
