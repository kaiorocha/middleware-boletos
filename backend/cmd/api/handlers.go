package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

type App struct {
	TenantSvc   *service.TenantService
	UserSvc     *service.UserService
	CustomerSvc *service.CustomerService
	ProviderSvc *service.ProviderService
	BoletoSvc   *service.BoletoService
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

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrDuplicateResource) {
		writeError(w, http.StatusConflict, "DUPLICATE_RESOURCE", err.Error())
		return
	}
	if errors.Is(err, service.ErrValidation) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error")
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/api/v1/tenants", a.handleTenants)
	mux.HandleFunc("/api/v1/users", a.handleUsers)
	mux.HandleFunc("/api/v1/tenants/", a.handleTenantsScoped)
	mux.HandleFunc("/api/v1/users/", a.handleUsersByID)
	return mux
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
	case "users":
		a.handleTenantUsers(w, r, tenantID, parts[2:])
	case "customers":
		a.handleTenantCustomers(w, r, tenantID, parts[2:])
	case "providers":
		a.handleTenantProviders(w, r, tenantID, parts[2:])
	case "boletos":
		a.handleTenantBoletos(w, r, tenantID, parts[2:])
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	}
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
		writeJSON(w, http.StatusOK, item)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
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
