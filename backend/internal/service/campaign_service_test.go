package service

import (
	"bytes"
	"testing"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
)

type campaignRepoFake struct {
	preview  domain.CampaignImportPreview
	rows     []domain.StagedCampaignRow
	existing map[string]bool
}

func (f *campaignRepoFake) Create(*domain.Campaign) error                { return nil }
func (f *campaignRepoFake) Get(string, string) (*domain.Campaign, error) { return nil, nil }
func (f *campaignRepoFake) Update(*domain.Campaign) error                { return nil }
func (f *campaignRepoFake) List(domain.CampaignFilters) ([]domain.Campaign, int, error) {
	return nil, 0, nil
}
func (f *campaignRepoFake) ExistingExternalIDs(string, []string) (map[string]bool, error) {
	return f.existing, nil
}
func (f *campaignRepoFake) SavePreview(_, _, _ string, rows []domain.StagedCampaignRow, p domain.CampaignImportPreview) (string, error) {
	f.rows = rows
	f.preview = p
	return "550e8400-e29b-41d4-a716-446655440099", nil
}
func (f *campaignRepoFake) ConfirmImport(string, string, string, string) (int, error) { return 0, nil }
func (f *campaignRepoFake) Start(string, string) error                                { return nil }
func (f *campaignRepoFake) Metrics(string, string) (*domain.CampaignMetrics, error) {
	return &domain.CampaignMetrics{}, nil
}
func (f *campaignRepoFake) Dashboard(string, string, string, string, *time.Time, *time.Time) (*domain.CampaignDashboard, error) {
	return &domain.CampaignDashboard{}, nil
}

func TestParseMoneyCents(t *testing.T) {
	tests := map[string]int64{"1499.00": 149900, "750.5": 75050, "0.01": 1, "12": 1200}
	for raw, want := range tests {
		got, err := parseMoneyCents(raw)
		if err != nil || got != want {
			t.Fatalf("parseMoneyCents(%q)=%d,%v want %d", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", "0", "-1", "1.001", "1,00", "1e2"} {
		if _, err := parseMoneyCents(raw); err == nil {
			t.Fatalf("expected %q to fail", raw)
		}
	}
}

func TestCampaignPreviewCSVValidAndInvalidRows(t *testing.T) {
	repo := &campaignRepoFake{existing: map[string]bool{"used": true}}
	svc := NewCampaignService(repo)
	campaign := &domain.Campaign{ID: "550e8400-e29b-41d4-a716-446655440001", TenantID: "550e8400-e29b-41d4-a716-446655440002", Status: domain.CampaignDraft}
	csv := `email,cpf_cnpj,valor,vencimento,external_id
 OK@Example.com ,529.982.247-25,1499.00,2099-09-30,new
bad,04.252.011/0001-10,10.00,2099-09-30,x
other@example.com,52998224725,2.50,2099-09-30,used
`
	p, err := svc.PreviewCSV(bytes.NewBufferString(csv), campaign, "user", "request")
	if err != nil {
		t.Fatal(err)
	}
	if p.TotalRows != 3 || p.ValidRows != 1 || p.InvalidRows != 2 || p.TotalAmountCents != 149900 {
		t.Fatalf("unexpected preview: %#v", p)
	}
	if repo.rows[0].Email != "ok@example.com" {
		t.Fatalf("email was not normalized: %q", repo.rows[0].Email)
	}
	if repo.rows[0].Document != "52998224725" {
		t.Fatalf("document was not normalized: %q", repo.rows[0].Document)
	}
	if p.ImportID == "" || p.Checksum == "" {
		t.Fatal("preview must be bound to persisted import and checksum")
	}
}

func TestCampaignPreviewCSVRejectsHeadersAndEmpty(t *testing.T) {
	svc := NewCampaignService(&campaignRepoFake{})
	c := &domain.Campaign{ID: "550e8400-e29b-41d4-a716-446655440001", TenantID: "550e8400-e29b-41d4-a716-446655440002", Status: domain.CampaignDraft}
	for _, csv := range []string{"foo,cpf_cnpj,valor,vencimento,external_id\n", "email,cpf_cnpj,valor,vencimento,external_id\n"} {
		if _, err := svc.PreviewCSV(bytes.NewBufferString(csv), c, "", ""); err == nil {
			t.Fatalf("expected invalid CSV %q", csv)
		}
	}
}

func TestCampaignPreviewCSVRejectsInvalidDocument(t *testing.T) {
	repo := &campaignRepoFake{}
	svc := NewCampaignService(repo)
	c := &domain.Campaign{ID: "550e8400-e29b-41d4-a716-446655440001", TenantID: "550e8400-e29b-41d4-a716-446655440002", Status: domain.CampaignDraft}
	p, err := svc.PreviewCSV(bytes.NewBufferString("email,cpf_cnpj,valor,vencimento,external_id\nuser@example.com,111.111.111-11,10.00,2099-09-30,row-1\n"), c, "", "")
	if err != nil { t.Fatal(err) }
	if p.ValidRows != 0 || p.InvalidRows != 1 || p.Errors[0].Code != "INVALID_DOCUMENT" { t.Fatalf("unexpected preview: %#v", p) }
}
