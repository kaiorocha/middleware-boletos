package base

import (
	"testing"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
)

func TestDefaultPayerBuilderNormalizesCustomer(t *testing.T) {
	customer := completeCustomer()

	payer, err := NewDefaultPayerBuilder().Build(customer)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if payer.Document != "12345678900" {
		t.Fatalf("expected normalized document, got %q", payer.Document)
	}
	if payer.PostalCode != "12345678" {
		t.Fatalf("expected normalized postal code, got %q", payer.PostalCode)
	}
	if payer.State != "SP" {
		t.Fatalf("expected uppercase state, got %q", payer.State)
	}
	if payer.Email != "cliente@example.com" {
		t.Fatalf("expected lowercase email, got %q", payer.Email)
	}
	if payer.Address != "Rua Um, 123, Apto 4" {
		t.Fatalf("expected full address, got %q", payer.Address)
	}
}

func TestDefaultPayerBuilderRequiredFields(t *testing.T) {
	tests := map[string]func(*customerFixture){
		"document":    func(c *customerFixture) { c.document = nil },
		"postal code": func(c *customerFixture) { c.postalCode = nil },
		"city":        func(c *customerFixture) { c.city = nil },
		"state":       func(c *customerFixture) { c.state = nil },
		"address":     func(c *customerFixture) { c.address = nil },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := completeCustomerFixture()
			mutate(&fixture)
			_, err := NewDefaultPayerBuilder().Build(fixture.customer())
			assertInvalidPayer(t, err)
		})
	}
}

func assertInvalidPayer(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected INVALID_PAYER error")
	}
	perr, ok := err.(*providererrors.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if perr.Code != invalidPayerCode {
		t.Fatalf("expected %s, got %s", invalidPayerCode, perr.Code)
	}
}

type customerFixture struct {
	name       string
	document   *string
	email      *string
	address    *string
	number     *string
	complement *string
	district   *string
	city       *string
	state      *string
	postalCode *string
}

func completeCustomerFixture() customerFixture {
	return customerFixture{
		name:       " Cliente Demo ",
		document:   strPtr("123.456.789-00"),
		email:      strPtr(" CLIENTE@EXAMPLE.COM "),
		address:    strPtr(" Rua Um "),
		number:     strPtr(" 123 "),
		complement: strPtr(" Apto 4 "),
		district:   strPtr(" Centro "),
		city:       strPtr(" Sao Paulo "),
		state:      strPtr(" sp "),
		postalCode: strPtr("12345-678"),
	}
}

func completeCustomer() domain.Customer {
	fixture := completeCustomerFixture()
	return fixture.customer()
}

func (f customerFixture) customer() domain.Customer {
	return domain.Customer{
		Name:       f.name,
		Document:   f.document,
		Email:      f.email,
		Address:    f.address,
		Number:     f.number,
		Complement: f.complement,
		District:   f.district,
		City:       f.city,
		State:      f.state,
		PostalCode: f.postalCode,
	}
}

func strPtr(value string) *string {
	return &value
}
