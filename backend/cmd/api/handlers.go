package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/contracts"
	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/webhooks"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type App struct {
	TenantSvc     *service.TenantService
	UserSvc       *service.UserService
	CustomerSvc   *service.CustomerService
	ProviderSvc   *service.ProviderService
	BoletoSvc     *service.BoletoService
	BlacklistSvc  *service.BlacklistService
	Factory       contracts.ProviderFactory
	Authorizer    TenantAuthorizer
	Authenticator *RequestAuthenticator
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": v})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeRawJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}
	if errors.Is(err, service.ErrCustomerBlocked) {
		writeError(w, http.StatusConflict, "CUSTOMER_BLOCKED", err.Error())
		return
	}
	if errors.Is(err, service.ErrDuplicateResource) {
		writeError(w, http.StatusConflict, "DUPLICATE_RESOURCE", err.Error())
		return
	}
	if errors.Is(err, service.ErrValidation) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	var providerErr *providererrors.ProviderError
	if errors.As(err, &providerErr) {
		switch providerErr.Code {
		case "INVALID_REQUEST", "INVALID_PAYER", "INVALID_PROVIDER_CONFIG", "PROVIDER_VALIDATION_ERROR":
			writeError(w, http.StatusBadRequest, providerErr.Code, providerErr.Message)
		case "UNSUPPORTED_OPERATION":
			writeError(w, http.StatusNotImplemented, providerErr.Code, providerErr.Message)
		default:
			writeError(w, http.StatusBadGateway, providerErr.Code, providerErr.Message)
		}
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error")
}

func (a *App) tenantAuthorizer() TenantAuthorizer {
	if a.Authorizer != nil {
		return a.Authorizer
	}
	return NewIdentityTenantAuthorizer()
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/api/v1/tenants", a.handleTenants)
	mux.HandleFunc("/api/v1/providers/", a.handleProvidersIntegration)
	mux.HandleFunc("/api/v1/users", a.handleUsers)
	mux.HandleFunc("/api/v1/tenants/", a.handleTenantsScoped)
	mux.HandleFunc("/api/v1/users/", a.handleUsersByID)
	return a.authenticationMiddleware(mux)
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleTenants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var in domain.Tenant
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
			return
		}
		if err := a.TenantSvc.Create(&in); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, in)
	case http.MethodGet:
		items, err := a.TenantSvc.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tenants")
			return
		}
		writeJSON(w, http.StatusOK, items)
	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	var in domain.User
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
		return
	}
	if !service.IsValidUUID(in.TenantID) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid tenant id")
		return
	}
	if decision := a.tenantAuthorizer().AuthorizeTenant(r, in.TenantID); !decision.Authenticated {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	} else if !decision.Allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}
	if err := a.UserSvc.Create(&in); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, in)
}

func (a *App) handleUsersByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	if !service.IsValidUUID(id) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user id")
		return
	}
	item, err := a.UserSvc.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if decision := a.tenantAuthorizer().AuthorizeTenant(r, item.TenantID); !decision.Authenticated {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	} else if !decision.Allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *App) handleTenantsScoped(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/")
	parts := splitPath(path)
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}

	tenantID := parts[0]
	if !service.IsValidUUID(tenantID) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid tenant id")
		return
	}
	if decision := a.tenantAuthorizer().AuthorizeTenant(r, tenantID); !decision.Authenticated {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	} else if !decision.Allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		item, err := a.TenantSvc.Get(tenantID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "tenant not found")
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}

	resource := parts[1]
	switch resource {
	case "dashboard":
		a.handleTenantDashboard(w, r, tenantID, parts[2:])
	case "users":
		a.handleTenantUsers(w, r, tenantID, parts[2:])
	case "customers":
		a.handleTenantCustomers(w, r, tenantID, parts[2:])
	case "providers":
		a.handleTenantProviders(w, r, tenantID, parts[2:])
	case "boletos":
		a.handleTenantBoletos(w, r, tenantID, parts[2:])
	case "blacklist":
		a.handleTenantBlacklist(w, r, tenantID, parts[2:])
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	}
}

func (a *App) handleTenantDashboard(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if len(tail) != 0 || r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	items, err := a.BoletoSvc.ListByTenant(tenantID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	var from, to time.Time
	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		from, _ = service.NormalizeDueDate(raw)
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("to")); raw != "" {
		to, _ = service.NormalizeDueDate(raw)
	}

	summary := map[string]any{
		"total_boletos":            0,
		"boletos_emitidos":         0,
		"boletos_em_processamento": 0,
		"boletos_pagos":            0,
		"boletos_vencidos":         0,
		"boletos_cancelados":       0,
		"boletos_com_falha":        0,
		"valor_total_emitido":      int64(0),
		"by_status":                map[string]int{},
	}
	byStatus := summary["by_status"].(map[string]int)
	for _, boleto := range items {
		if !from.IsZero() && boleto.CreatedAt.Before(from) {
			continue
		}
		if !to.IsZero() && boleto.CreatedAt.After(to.Add(24*time.Hour)) {
			continue
		}
		summary["total_boletos"] = summary["total_boletos"].(int) + 1
		byStatus[boleto.Status]++
		switch types.BoletoStatus(boleto.Status) {
		case types.StatusIssued:
			summary["boletos_emitidos"] = summary["boletos_emitidos"].(int) + 1
			summary["valor_total_emitido"] = summary["valor_total_emitido"].(int64) + boleto.AmountCents
		case types.StatusProcessing:
			summary["boletos_em_processamento"] = summary["boletos_em_processamento"].(int) + 1
		case types.StatusPaid:
			summary["boletos_pagos"] = summary["boletos_pagos"].(int) + 1
		case types.StatusExpired:
			summary["boletos_vencidos"] = summary["boletos_vencidos"].(int) + 1
		case types.StatusCancelled:
			summary["boletos_cancelados"] = summary["boletos_cancelados"].(int) + 1
		case types.StatusFailed:
			summary["boletos_com_falha"] = summary["boletos_com_falha"].(int) + 1
		}
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *App) handleTenantUsers(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if len(tail) != 0 || r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	items, err := a.UserSvc.ListByTenant(tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) handleTenantCustomers(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodPost:
			var in domain.Customer
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
				return
			}
			in.TenantID = tenantID
			if err := a.CustomerSvc.Create(&in); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, in)
		case http.MethodGet:
			items, err := a.CustomerSvc.ListByTenant(tenantID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list customers")
				return
			}
			writeJSON(w, http.StatusOK, items)
		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	if len(tail) == 1 {
		id := tail[0]
		if !service.IsValidUUID(id) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid customer id")
			return
		}
		switch r.Method {
		case http.MethodGet:
			item, err := a.CustomerSvc.Get(id)
			if err != nil || item.TenantID != tenantID {
				writeError(w, http.StatusNotFound, "NOT_FOUND", "customer not found")
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodPut:
			var in domain.Customer
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
				return
			}
			in.ID = id
			in.TenantID = tenantID
			if err := a.CustomerSvc.Update(&in); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, in)
		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
}

func (a *App) handleTenantProviders(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodPost:
			var in domain.Provider
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
				return
			}
			in.TenantID = tenantID
			if err := a.ProviderSvc.Create(&in); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, in)
		case http.MethodGet:
			items, err := a.ProviderSvc.ListByTenant(tenantID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list providers")
				return
			}
			for i := range items {
				maskProviderConfig(&items[i])
			}
			writeJSON(w, http.StatusOK, items)
		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	if len(tail) == 1 && r.Method == http.MethodGet {
		id := tail[0]
		if !service.IsValidUUID(id) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid provider id")
			return
		}
		item, err := a.ProviderSvc.Get(id)
		if err != nil || item.TenantID != tenantID {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "provider not found")
			return
		}
		maskProviderConfig(item)
		writeJSON(w, http.StatusOK, item)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}

func (a *App) handleTenantBlacklist(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if a.BlacklistSvc == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "blacklist service not configured")
		return
	}
	if len(tail) == 1 && tail[0] == "check" && r.Method == http.MethodGet {
		document := r.URL.Query().Get("document")
		entry, blocked, err := a.BlacklistSvc.IsBlocked(tenantID, document)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if !blocked {
			writeRawJSON(w, http.StatusOK, map[string]bool{"blocked": false})
			return
		}
		writeRawJSON(w, http.StatusOK, map[string]any{"blocked": true, "reason": entry.Reason})
		return
	}

	if len(tail) == 0 {
		switch r.Method {
		case http.MethodPost:
			var in domain.BlacklistEntry
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
				return
			}
			in.TenantID = tenantID
			if err := a.BlacklistSvc.Create(&in); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, in)
		case http.MethodGet:
			active, err := parseOptionalBool(r.URL.Query().Get("active"))
			if err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid active filter")
				return
			}
			items, err := a.BlacklistSvc.List(tenantID, r.URL.Query().Get("q"), active)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, items)
		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	id := tail[0]
	if !service.IsValidUUID(id) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid blacklist id")
		return
	}
	if len(tail) == 2 {
		switch tail[1] {
		case "block":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
				return
			}
			item, err := a.BlacklistSvc.Block(tenantID, id, nil)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
			return
		case "unblock":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
				return
			}
			item, err := a.BlacklistSvc.Unblock(tenantID, id, nil)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
			return
		default:
			writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
			return
		}
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			item, err := a.BlacklistSvc.Get(tenantID, id)
			if err != nil {
				writeError(w, http.StatusNotFound, "NOT_FOUND", "blacklist entry not found")
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodPut:
			var in domain.BlacklistEntry
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
				return
			}
			in.ID = id
			in.TenantID = tenantID
			if err := a.BlacklistSvc.Update(&in); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, in)
		case http.MethodDelete:
			if err := a.BlacklistSvc.Delete(tenantID, id); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}
	writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
}

func (a *App) handleProvidersIntegration(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/providers/")
	parts := splitPath(path)
	if len(parts) != 1 {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}
	if a.Factory == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "provider factory not configured")
		return
	}

	switch parts[0] {
	case "health":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		adapter, err := a.providerAdapterFromRequest(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		health, err := adapter.Health(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "PROVIDER_ERROR", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, health)
	case "balance":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		cfg, adapter, err := a.providerConfigAndAdapterFromRequest(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		balance, err := adapter.GetBalance(r.Context(), types.BalanceRequest{TenantID: cfg.TenantID, ProviderID: cfg.ID})
		if err != nil {
			writeError(w, http.StatusBadGateway, "PROVIDER_ERROR", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, balance)
	case "webhook":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		cfg, adapter, err := a.providerConfigAndAdapterFromRequest(r)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
			return
		}
		event, err := webhooks.Receive(r.Context(), adapter, types.ValidateWebhookRequest{ProviderID: cfg.ID, Headers: requestHeaders(r), Body: body})
		if err != nil {
			writeError(w, http.StatusBadRequest, "WEBHOOK_VALIDATION_ERROR", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, event)
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	}
}

func (a *App) providerAdapterFromRequest(r *http.Request) (contracts.ProviderAdapter, error) {
	_, adapter, err := a.providerConfigAndAdapterFromRequest(r)
	return adapter, err
}

func (a *App) providerConfigAndAdapterFromRequest(r *http.Request) (types.ProviderConfig, contracts.ProviderAdapter, error) {
	query := r.URL.Query()
	tenantID := query.Get("tenant_id")
	providerID := query.Get("provider_id")
	cfg := types.ProviderConfig{ID: providerID, TenantID: tenantID, Name: "Mock"}

	if providerID != "" {
		if !service.IsValidUUID(providerID) || !service.IsValidUUID(tenantID) || a.ProviderSvc == nil {
			return cfg, nil, service.ErrValidation
		}
		if decision := a.tenantAuthorizer().AuthorizeTenant(r, tenantID); !decision.Authenticated {
			return cfg, nil, ErrUnauthorized
		} else if !decision.Allowed {
			return cfg, nil, ErrForbidden
		}
		provider, err := a.ProviderSvc.Get(providerID)
		if err != nil {
			return cfg, nil, err
		}
		if provider.TenantID != tenantID {
			return cfg, nil, service.ErrValidation
		}
		cfg = types.ProviderConfig{ID: provider.ID, TenantID: provider.TenantID, Name: provider.Name}
		if provider.Config != nil {
			cfg.Config = *provider.Config
		}
	}

	adapter, err := a.Factory.Build(cfg)
	return cfg, adapter, err
}

func requestHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string, len(r.Header))
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

func parseOptionalBool(raw string) (*bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func maskProviderConfig(provider *domain.Provider) {
	if provider == nil || provider.Config == nil {
		return
	}
	masked := "***"
	provider.Config = &masked
}

func requestID(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Request-ID")); id != "" {
		return id
	}
	return uuid.New().String()
}

func (a *App) handleTenantBoletos(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodPost:
			var in struct {
				CustomerID    string  `json:"customer_id"`
				ProviderID    *string `json:"provider_id"`
				AmountCents   int64   `json:"amount_cents"`
				DueDate       string  `json:"due_date"`
				Status        string  `json:"status"`
				ExternalID    *string `json:"external_id"`
				Barcode       *string `json:"barcode"`
				DigitableLine *string `json:"digitable_line"`
				OurNumber     *string `json:"our_number"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
				return
			}
			dueDate, err := service.NormalizeDueDate(in.DueDate)
			if err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid due_date, expected YYYY-MM-DD")
				return
			}
			item := domain.Boleto{
				TenantID:      tenantID,
				CustomerID:    in.CustomerID,
				ProviderID:    in.ProviderID,
				AmountCents:   in.AmountCents,
				DueDate:       dueDate,
				Status:        in.Status,
				ExternalID:    in.ExternalID,
				Barcode:       in.Barcode,
				DigitableLine: in.DigitableLine,
				OurNumber:     in.OurNumber,
			}
			if err := a.BoletoSvc.Create(&item); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, item)
		case http.MethodGet:
			items, err := a.BoletoSvc.ListByTenant(tenantID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list boletos")
				return
			}
			writeJSON(w, http.StatusOK, items)
		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	if len(tail) == 2 && tail[1] == "emit" && r.Method == http.MethodPost {
		id := tail[0]
		if !service.IsValidUUID(id) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid boleto id")
			return
		}
		ctx := service.WithRequestID(r.Context(), requestID(r))
		item, err := a.BoletoSvc.Emit(ctx, tenantID, id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}

	if len(tail) == 1 && r.Method == http.MethodGet {
		id := tail[0]
		if !service.IsValidUUID(id) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid boleto id")
			return
		}
		item, err := a.BoletoSvc.Get(id)
		if err != nil || item.TenantID != tenantID {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "boleto not found")
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
