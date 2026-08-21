package base

import (
	"strings"
	"unicode"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

const invalidPayerCode = "INVALID_PAYER"
const invalidPayerMessage = "Customer does not contain all required address information for provider emission."

type PayerBuilder interface {
	Build(customer domain.Customer) (*types.Payer, error)
}

type DefaultPayerBuilder struct{}

func NewDefaultPayerBuilder() *DefaultPayerBuilder {
	return &DefaultPayerBuilder{}
}

func (b *DefaultPayerBuilder) Build(customer domain.Customer) (*types.Payer, error) {
	name := strings.TrimSpace(customer.Name)
	document := digitsFromPtr(customer.Document)
	baseAddress := trimPtr(customer.Address)
	address := fullAddress(customer.Address, customer.Number, customer.Complement)
	district := trimPtr(customer.District)
	city := trimPtr(customer.City)
	state := strings.ToUpper(trimPtr(customer.State))
	postalCode := digitsFromPtr(customer.PostalCode)
	email := strings.ToLower(trimPtr(customer.Email))

	if name == "" || document == "" || baseAddress == "" || address == "" || district == "" || city == "" || state == "" || postalCode == "" {
		return nil, invalidPayerError()
	}
	if len(state) != 2 || len(postalCode) != 8 {
		return nil, invalidPayerError()
	}

	return &types.Payer{
		Document:   document,
		Name:       name,
		Address:    address,
		District:   district,
		City:       city,
		PostalCode: postalCode,
		State:      state,
		Email:      email,
	}, nil
}

func invalidPayerError() error {
	return providererrors.New(invalidPayerCode, invalidPayerMessage, "PayerBuilder", false)
}

func fullAddress(address, number, complement *string) string {
	parts := []string{trimPtr(address), trimPtr(number), trimPtr(complement)}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, ", ")
}

func trimPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func digitsFromPtr(value *string) string {
	if value == nil {
		return ""
	}
	var b strings.Builder
	for _, r := range *value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
