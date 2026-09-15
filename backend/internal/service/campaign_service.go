package service

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

const CampaignCSVMaxBytes int64 = 16 << 20
const CampaignCSVMaxRows = 100000
const CampaignCSVErrorLimit = 200

type campaignRepository interface {
	Create(*domain.Campaign) error
	Get(string, string) (*domain.Campaign, error)
	Update(*domain.Campaign) error
	List(domain.CampaignFilters) ([]domain.Campaign, int, error)
	ExistingExternalIDs(string, []string) (map[string]bool, error)
	SavePreview(string, string, string, []domain.StagedCampaignRow, domain.CampaignImportPreview) (string, error)
	ConfirmImport(string, string, string, string) (int, error)
	Start(string, string) error
	Metrics(string, string) (*domain.CampaignMetrics, error)
	Dashboard(string, string, string, string, *time.Time, *time.Time) (*domain.CampaignDashboard, error)
}
type auditWriter interface{ Create(*domain.AuditLog) error }

type CampaignService struct {
	repo  campaignRepository
	audit auditWriter
}

func NewCampaignService(repo campaignRepository) *CampaignService {
	return &CampaignService{repo: repo}
}
func (s *CampaignService) WithAuditRepository(a auditWriter) *CampaignService { s.audit = a; return s }

func (s *CampaignService) auditEvent(c *domain.Campaign, userID, action, requestID string, extra map[string]any) {
	if s.audit == nil {
		return
	}
	extra["campaign_id"] = c.ID
	extra["request_id"] = requestID
	b, _ := json.Marshal(extra)
	var uid *string
	if userID != "" {
		uid = &userID
	}
	metadata := string(b)
	_ = s.audit.Create(&domain.AuditLog{TenantID: c.TenantID, UserID: uid, Action: action, Metadata: &metadata})
}

func (s *CampaignService) Create(c *domain.Campaign, userID, requestID string) error {
	c.Name = strings.TrimSpace(c.Name)
	if !IsValidUUID(c.TenantID) || c.Name == "" || len(c.Name) > 160 {
		return ErrValidation
	}
	if c.Description != nil {
		v := strings.TrimSpace(*c.Description)
		if v == "" {
			c.Description = nil
		} else if len(v) > 2000 {
			return ErrValidation
		} else {
			c.Description = &v
		}
	}
	c.Status = domain.CampaignDraft
	c.CreatedBy = &userID
	if err := s.repo.Create(c); err != nil {
		return err
	}
	s.auditEvent(c, userID, "CampaignCreated", requestID, map[string]any{})
	return nil
}
func (s *CampaignService) Get(t, id string) (*domain.Campaign, error) {
	if !IsValidUUID(t) || !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.Get(t, id)
}
func (s *CampaignService) Update(c *domain.Campaign, userID, requestID string) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" || len(c.Name) > 160 {
		return ErrValidation
	}
	if err := s.repo.Update(c); err != nil {
		return err
	}
	s.auditEvent(c, userID, "CampaignUpdated", requestID, map[string]any{})
	return nil
}
func (s *CampaignService) List(f domain.CampaignFilters) ([]domain.Campaign, int, error) {
	if !IsValidUUID(f.TenantID) {
		return nil, 0, ErrValidation
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.repo.List(f)
}

func parseMoneyCents(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("required")
	}
	if strings.ContainsAny(raw, ",eE+") {
		return 0, errors.New("invalid")
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || len(parts[0]) == 0 {
		return 0, errors.New("invalid")
	}
	if len(parts) == 2 && len(parts[1]) > 2 {
		return 0, errors.New("invalid")
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 {
		return 0, errors.New("invalid")
	}
	frac := int64(0)
	if len(parts) == 2 {
		f := parts[1]
		if len(f) == 1 {
			f += "0"
		}
		if f == "" {
			f = "00"
		}
		frac, err = strconv.ParseInt(f, 10, 64)
		if err != nil {
			return 0, errors.New("invalid")
		}
	}
	if whole > (1<<63-1-frac)/100 {
		return 0, errors.New("invalid")
	}
	v := whole*100 + frac
	if v <= 0 {
		return 0, errors.New("invalid")
	}
	return v, nil
}

func (s *CampaignService) PreviewCSV(reader io.Reader, c *domain.Campaign, userID, requestID string) (*domain.CampaignImportPreview, error) {
	if c.Status != domain.CampaignDraft {
		return nil, ErrValidation
	}
	h := sha256.New()
	tee := io.TeeReader(io.LimitReader(reader, CampaignCSVMaxBytes+1), h)
	br := bufio.NewReader(tee)
	csvr := csv.NewReader(br)
	csvr.FieldsPerRecord = -1
	csvr.TrimLeadingSpace = true
	head, err := csvr.Read()
	if err != nil {
		return nil, ErrValidation
	}
	expected := []string{"email", "valor", "vencimento", "external_id"}
	if len(head) != len(expected) {
		return nil, NewValidation("Cabeçalhos inválidos.")
	}
	for i := range expected {
		if strings.ToLower(strings.TrimSpace(strings.TrimPrefix(head[i], "\ufeff"))) != expected[i] {
			return nil, NewValidation("Cabeçalhos inválidos.")
		}
	}
	rows := make([]domain.StagedCampaignRow, 0, 1024)
	external := []string{}
	seen := map[string]int{}
	p := domain.CampaignImportPreview{ErrorLimit: CampaignCSVErrorLimit}
	line := 1
	for {
		record, e := csvr.Read()
		if e == io.EOF {
			break
		}
		line++
		if e != nil {
			return nil, NewValidation("CSV inválido.")
		}
		p.TotalRows++
		if p.TotalRows > CampaignCSVMaxRows {
			return nil, NewValidation("O CSV excede 100.000 registros.")
		}
		row := domain.StagedCampaignRow{Row: line}
		valid := true
		fail := func(field, code, msg string) {
			if valid {
				row.Field, row.Code, row.Message = field, code, msg
			}
			valid = false
		}
		if len(record) != 4 {
			fail("row", "INVALID_COLUMNS", "A linha deve conter quatro colunas.")
		} else {
			row.Email = NormalizeEmail(record[0])
			if !IsValidEmail(row.Email) {
				fail("email", "INVALID_EMAIL", "Email inválido.")
			}
			row.Amount, e = parseMoneyCents(record[1])
			if e != nil {
				fail("valor", "INVALID_AMOUNT", "Valor inválido.")
			}
			row.DueDate, e = NormalizeDueDate(strings.TrimSpace(record[2]))
			if e != nil || row.DueDate.Before(time.Now().UTC().Truncate(24*time.Hour)) {
				fail("vencimento", "INVALID_DUE_DATE", "Vencimento inválido.")
			}
			ext := strings.TrimSpace(record[3])
			if ext != "" {
				row.ExternalID = &ext
				if first, ok := seen[ext]; ok {
					fail("external_id", "DUPLICATE_EXTERNAL_ID", fmt.Sprintf("external_id duplicado (primeira ocorrência na linha %d).", first))
				} else {
					seen[ext] = line
					external = append(external, ext)
				}
			}
		}
		if valid {
			p.ValidRows++
			p.TotalAmountCents += row.Amount
		} else {
			p.InvalidRows++
			if len(p.Errors) < p.ErrorLimit {
				p.Errors = append(p.Errors, domain.CampaignImportError{Row: line, Field: row.Field, Code: row.Code, Message: row.Message})
			}
		}
		rows = append(rows, row)
	}
	if p.TotalRows == 0 {
		return nil, NewValidation("O CSV não contém registros.")
	}
	if _, e := br.Peek(1); e == nil {
		return nil, NewValidation("O arquivo excede 16 MiB.")
	}
	existing, err := s.repo.ExistingExternalIDs(c.TenantID, external)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].ExternalID != nil && existing[*rows[i].ExternalID] && rows[i].Code == "" {
			rows[i].Field = "external_id"
			rows[i].Code = "DUPLICATE_EXTERNAL_ID"
			rows[i].Message = "external_id já existe neste tenant."
			p.ValidRows--
			p.InvalidRows++
			p.TotalAmountCents -= rows[i].Amount
			if len(p.Errors) < p.ErrorLimit {
				p.Errors = append(p.Errors, domain.CampaignImportError{Row: rows[i].Row, Field: rows[i].Field, Code: rows[i].Code, Message: rows[i].Message})
			}
		}
	}
	p.Checksum = hex.EncodeToString(h.Sum(nil))
	p.ImportID, err = s.repo.SavePreview(c.ID, c.TenantID, p.Checksum, rows, p)
	if err != nil {
		return nil, err
	}
	s.auditEvent(c, userID, "CampaignCSVUploaded", requestID, map[string]any{"total_rows": p.TotalRows, "valid_rows": p.ValidRows, "invalid_rows": p.InvalidRows, "total_amount_cents": p.TotalAmountCents, "checksum": p.Checksum})
	return &p, nil
}
func (s *CampaignService) Confirm(c *domain.Campaign, importID, providerID, userID, requestID string) (int, error) {
	if c.Status != domain.CampaignDraft || !IsValidUUID(importID) || !IsValidUUID(providerID) {
		return 0, ErrValidation
	}
	n, err := s.repo.ConfirmImport(c.TenantID, c.ID, importID, providerID)
	if err == nil {
		s.auditEvent(c, userID, "CampaignImportConfirmed", requestID, map[string]any{"quantity": n})
	}
	return n, err
}
func (s *CampaignService) Start(c *domain.Campaign, userID, requestID string) error {
	if c.Status != domain.CampaignReady {
		return ErrValidation
	}
	metrics, err := s.repo.Metrics(c.TenantID, c.ID)
	if err != nil {
		return err
	}
	if err := s.repo.Start(c.TenantID, c.ID); err != nil {
		return err
	}
	s.auditEvent(c, userID, "CampaignIssuanceStarted", requestID, map[string]any{"quantity": metrics.Waiting, "total_amount_cents": metrics.ByStatusAmount("CREATED")})
	return nil
}
func (s *CampaignService) Metrics(t, id string) (*domain.CampaignMetrics, error) {
	if !IsValidUUID(t) || !IsValidUUID(id) {
		return nil, ErrValidation
	}
	return s.repo.Metrics(t, id)
}
func (s *CampaignService) Dashboard(tenant, campaign, provider, status string, from, to *time.Time) (*domain.CampaignDashboard, error) {
	for _, id := range []string{tenant, campaign, provider} {
		if id != "" && !IsValidUUID(id) {
			return nil, ErrValidation
		}
	}
	return s.repo.Dashboard(tenant, campaign, provider, strings.ToUpper(strings.TrimSpace(status)), from, to)
}

type ValidationError struct{ message string }

func (e ValidationError) Error() string { return e.message }
func (e ValidationError) Unwrap() error { return ErrValidation }
func NewValidation(m string) error      { return ValidationError{m} }

type campaignIssuanceRepository interface {
	ProcessingCampaigns(int) ([]domain.Campaign, error)
	ClaimPendingBoletoIDs(string, int) ([]string, error)
	FinishIfDone(string) error
	MarkBoletoTerminal(string, string) error
}

type CampaignIssuer struct {
	repo   campaignIssuanceRepository
	boleto *BoletoService
}

func NewCampaignIssuer(r campaignIssuanceRepository, b *BoletoService) *CampaignIssuer {
	return &CampaignIssuer{repo: r, boleto: b}
}
func (i *CampaignIssuer) ProcessPending(ctx context.Context, limit int) (int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	campaigns, err := i.repo.ProcessingCampaigns(10)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, c := range campaigns {
		ids, err := i.repo.ClaimPendingBoletoIDs(c.ID, limit)
		if err != nil {
			return processed, err
		}
		for _, id := range ids {
			_, issueErr := i.boleto.Emit(ctx, c.TenantID, id)
			if issueErr != nil {
				status := "FAILED"
				if errors.Is(issueErr, ErrRecipientBlocked) || errors.Is(issueErr, ErrCustomerBlocked) {
					status = "BLOCKED"
				}
				_ = i.repo.MarkBoletoTerminal(id, status)
			}
			processed++
		}
		if err = i.repo.FinishIfDone(c.ID); err != nil {
			return processed, err
		}
		if processed >= limit {
			break
		}
	}
	return processed, nil
}
