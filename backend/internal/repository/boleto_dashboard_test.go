package repository

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	_ "github.com/lib/pq"
)

// Run with TEST_DATABASE_URL pointing to a disposable PostgreSQL database.
func TestSettlementDashboardPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	exec := func(query string) {
		t.Helper()
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	// Use a private schema: PostgreSQL does not resolve functions in pg_temp.
	schema := fmt.Sprintf("dashboard_test_%d", time.Now().UnixNano())
	exec("CREATE SCHEMA " + schema)
	defer func() {
		if _, err := db.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Errorf("cleanup test schema: %v", err)
		}
	}()
	exec("SET search_path TO " + schema)
	exec(`CREATE TABLE tenants(id text, name text, deleted_at timestamptz);
 CREATE TABLE customers(id text, document text, email text);
 CREATE TABLE providers(id text, name text);
 CREATE TABLE boletos(id text, tenant_id text, customer_id text, provider_id text, status text, amount_cents bigint, created_at timestamptz, deleted_at timestamptz);
 INSERT INTO tenants VALUES ('a','Tenant A',NULL),('b','Tenant B',NULL);
 INSERT INTO boletos(id,tenant_id,status,amount_cents,created_at) VALUES
 ('historical','a','PAID',1000,'2026-09-01'),
 ('pending','a','ISSUED',2000,'2026-09-01'),
 ('failed','a','FAILED',9000,'2026-09-01');`)
	migration, err := os.ReadFile("../storage/migrations/028_track_boleto_payment_confirmation.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(migration))
	exec(`INSERT INTO boletos(id,tenant_id,status,amount_cents,created_at,paid_at) VALUES
 ('paid','a','PAID',2000,'2026-09-01','2026-09-03 01:00:00+00'),
 ('other','b','PAID',8000,'2026-09-01','2026-09-03 01:00:00+00'),
 ('old','a','PAID',7000,'2026-08-01','2026-09-03 01:00:00+00'),
 ('deleted','a','PAID',6000,'2026-09-01','2026-09-03 01:00:00+00');
 UPDATE boletos SET deleted_at=now() WHERE id='deleted';
 UPDATE boletos SET amount_cents=1000 WHERE id='historical';`)
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	dash, err := NewBoletoRepo(db).AdminDashboard(domain.BoletoFilters{TenantID: "a", From: &from, To: &to})
	if err != nil {
		t.Fatal(err)
	}
	s := dash.Settlement
	if s.Issued != 3 || s.Paid != 2 || s.PaidAmountCents != 3000 || s.UnknownDateCount != 1 {
		t.Fatalf("unexpected settlement totals: %+v", s)
	}
	if len(s.Timeline) != 1 || s.Timeline[0].Date != "2026-09-02" || s.Timeline[0].Count != 1 || s.Timeline[0].AmountCents != 2000 {
		t.Fatalf("unexpected confirmation dates: %+v", s.Timeline)
	}
	if len(s.ByAmount) != 2 || s.ByAmount[0].ID != "1000" || s.ByAmount[1].AmountCents != 2000 {
		t.Fatalf("unexpected amounts: %+v", s.ByAmount)
	}
	exec(`UPDATE boletos SET status='PAID' WHERE id='pending'`)
	var first, again time.Time
	if err := db.QueryRow(`SELECT paid_at FROM boletos WHERE id='pending'`).Scan(&first); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE boletos SET amount_cents=2100 WHERE id='pending'`)
	if err := db.QueryRow(`SELECT paid_at FROM boletos WHERE id='pending'`).Scan(&again); err != nil {
		t.Fatal(err)
	}
	if !first.Equal(again) {
		t.Fatal("repeated updates moved payment confirmation date")
	}
	empty, err := NewBoletoRepo(db).AdminDashboard(domain.BoletoFilters{TenantID: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if empty.Settlement.Paid != 0 || len(empty.Settlement.Timeline) != 0 || len(empty.Settlement.ByAmount) != 0 {
		t.Fatalf("unexpected empty dashboard: %+v", empty)
	}
}
