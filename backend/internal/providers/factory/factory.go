package factory

import (
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/contracts"
	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/mock"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/moncalieri"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

type ProviderFactory struct{}

func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

func (f *ProviderFactory) Build(cfg types.ProviderConfig) (contracts.ProviderAdapter, error) {
	switch normalize(cfg.Name) {
	case "mock":
		return mock.New(cfg), nil
	case "moncalieri", "moncaliericapital":
		return moncalieri.New(cfg), nil
	case "bancox", "bancoy":
		return nil, providererrors.New("PROVIDER_NOT_IMPLEMENTED", "provider adapter is not implemented yet", cfg.Name, false)
	default:
		return nil, providererrors.New("PROVIDER_NOT_FOUND", "provider adapter not found", cfg.Name, false)
	}
}

func normalize(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
}
