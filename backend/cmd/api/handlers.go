package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"log/slog"

	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/contracts"
	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/webhooks"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

const defaultTokenTTL = time.Hour

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type dbPinger interface {
	PingContext(context.Context) error
}

type App struct {
	DB            dbPinger
	TenantSvc     *service.TenantService
	UserSvc       *service.UserService
	CustomerSvc   *service.CustomerService
	ProviderSvc   *service.ProviderService
	BoletoSvc     *service.BoletoService
	BlacklistSvc  *service.BlacklistService
	OnboardingSvc *service.OnboardingService
	Factory       contracts.ProviderFactory
	Authorizer    TenantAuthorizer
	Authenticator *RequestAuthenticator
	TokenIssuer   authn.TokenIssuer
	CORSOrigins   []string
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
	if errors.Is(err, service.ErrRecipientBlocked) {
		writeError(w, http.StatusConflict, "RECIPIENT_BLOCKED", err.Error())
		return
	}
	if errors.Is(err, service.ErrProviderNotAllowed) {
		writeError(w, http.StatusForbidden, "PROVIDER_NOT_ALLOWED", err.Error())
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
	mux.HandleFunc("/ready", a.handleReady)
	mux.HandleFunc("/api/v1/auth/login", a.handleLogin)
	mux.HandleFunc("/api/v1/tenants", a.handleTenants)
	mux.HandleFunc("/api/v1/me/tenants", a.handleMyTenants)
	mux.HandleFunc("/api/v1/admin/dashboard", a.handleAdminDashboard)
	mux.HandleFunc("/api/v1/admin/transactions", a.handleAdminTransactions)
	mux.HandleFunc("/api/v1/admin/providers", a.handleAdminProviders)
	mux.HandleFunc("/api/v1/admin/providers/", a.handleAdminProviderByID)
	mux.HandleFunc("/api/v1/admin/tenants", a.handleAdminTenants)
	mux.HandleFunc("/api/v1/providers/", a.handleProvidersIntegration)
	mux.HandleFunc("/api/v1/users", a.handleUsers)
	mux.HandleFunc("/api/v1/tenants/", a.handleTenantsScoped)
	mux.HandleFunc("/api/v1/users/", a.handleUsersByID)

	// base handler with auth and CORS
	h := corsMiddleware(a.authenticationMiddleware(mux), a.CORSOrigins)

	// apply security / request protections
	h = a.requestBodyLimitMiddleware(h)
	h = a.securityHeadersMiddleware(h)

	// request id should be available early
	h = a.requestIDMiddleware(h)

	// access log after request id
	h = a.accessLogMiddleware(h)

	// recovery outermost
	h = a.recoveryMiddleware(h)

	return h
}

func corsMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	allowed := map[string]struct{}{}
	allowAny := len(allowedOrigins) == 0
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "*" {
			allowAny = true
		}
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if allowAny {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	if a.DB == nil {
		writeError(w, http.StatusServiceUnavailable, "NOT_READY", "service not ready")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.DB.PingContext(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "NOT_READY", "service not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// --- Middleware and helpers ---

type ctxKey string

const requestIDKey ctxKey = "request_id"

func (a *App) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		valid := false
		if rid != "" && len(rid) <= 128 {
			// accept only UUID values
			if _, err := uuid.Parse(rid); err == nil {
				valid = true
			}
		}
		if !valid {
			rid = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, rid)))
	})
}

// responseWriter wrapper to capture status

type responseWriter struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.w.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.w.Write(b)
	rw.size += n
	return n, err
}

func (a *App) accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{w: w}
		next.ServeHTTP(rw, r)
		dur := time.Since(start)
		requestID, _ := r.Context().Value(requestIDKey).(string)
		logger := slog.Default()
		logger.Info("request_completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", dur.Milliseconds(),
			"request_id", requestID,
		)
	})
}

func (a *App) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID, _ := r.Context().Value(requestIDKey).(string)
				logger := slog.Default()
				logger.Error("panic_recovered", "error", rec, "request_id", requestID)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (a *App) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (a *App) requestBodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// limit to 1MB
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleTenants(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlatformAdmin(w, r) {
		return
	}
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

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	if a.UserSvc == nil || a.TokenIssuer == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "authentication not configured")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
		return
	}
	user, err := a.UserSvc.GetByEmail(in.Email)
	if err != nil || user.PasswordHash == "" || !authn.ComparePassword(user.PasswordHash, in.Password) || user.Status != "ACTIVE" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Credenciais inválidas.")
		return
	}
	tenantIDs := tenantIDsForUser(user)
	token, err := a.TokenIssuer.Sign(authn.TokenClaims{
		UserID:    user.ID,
		TenantIDs: tenantIDs,
		Roles:     user.Roles,
		ExpiresAt: time.Now().Add(defaultTokenTTL),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   int(defaultTokenTTL.Seconds()),
		"user":         publicUser(user, tenantIDs),
	})
}

func (a *App) handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlatformAdmin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	var in struct {
		Name  string `json:"name"`
		Admin *struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"admin"`
		Providers []struct {
			ProviderID string  `json:"provider_id"`
			Active     *bool   `json:"active"`
			Config     *string `json:"config"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
		return
	}
	var admin *domain.User
	if in.Admin != nil && strings.TrimSpace(in.Admin.Email) != "" {
		if len(strings.TrimSpace(in.Admin.Password)) < 8 {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid admin password")
			return
		}
		hash, err := authn.HashPassword(in.Admin.Password)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid admin password")
			return
		}
		admin = &domain.User{
			Email:        in.Admin.Email,
			Name:         in.Admin.Name,
			Status:       "ACTIVE",
			Roles:        []string{authn.RoleTenantAdmin},
			PasswordHash: hash,
		}
	}
	providers := []domain.OnboardingProviderInput{}
	for _, provider := range in.Providers {
		active := true
		if provider.Active != nil {
			active = *provider.Active
		}
		providers = append(providers, domain.OnboardingProviderInput{ProviderID: provider.ProviderID, Active: active, Config: provider.Config})
	}
	if a.OnboardingSvc != nil {
		result, err := a.OnboardingSvc.CreateTenant(domain.OnboardingInput{
			Tenant:    domain.Tenant{Name: in.Name},
			Admin:     admin,
			Providers: providers,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"tenant": result.Tenant, "admin": publicUser(result.Admin, tenantIDsForUser(result.Admin)), "providers": maskTenantProviders(result.Providers)})
		return
	}

	tenant := &domain.Tenant{Name: in.Name}
	if err := a.TenantSvc.Create(tenant); err != nil {
		writeServiceError(w, err)
		return
	}
	if admin != nil {
		admin.TenantID = tenant.ID
		if err := a.UserSvc.Create(admin); err != nil {
			_ = a.TenantSvc.Delete(tenant.ID)
			writeServiceError(w, err)
			return
		}
	}
	assignments := []domain.TenantProvider{}
	for _, provider := range providers {
		assignment, err := a.ProviderSvc.AssignToTenant(tenant.ID, provider.ProviderID, provider.Active, provider.Config)
		if err != nil {
			_ = a.TenantSvc.Delete(tenant.ID)
			writeServiceError(w, err)
			return
		}
		assignments = append(assignments, *assignment)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"tenant": tenant, "admin": publicUser(admin, tenantIDsForUser(admin)), "providers": maskTenantProviders(assignments)})
}

func (a *App) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlatformAdmin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	filters, err := parseBoletoFilters(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	dashboard, err := a.BoletoSvc.AdminDashboard(filters)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (a *App) handleAdminTransactions(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlatformAdmin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	filters, err := parseBoletoFilters(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filters.Limit = parseIntQuery(r, "limit", 50)
	filters.Offset = parseIntQuery(r, "offset", 0)
	transactions, err := a.BoletoSvc.ListTransactions(filters)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, transactions)
}

func (a *App) handleAdminProviders(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlatformAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodPost:
		var in domain.Provider
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
			return
		}
		if err := a.ProviderSvc.CreateCatalog(&in); err != nil {
			writeServiceError(w, err)
			return
		}
		maskProviderConfig(&in)
		writeJSON(w, http.StatusCreated, in)
	case http.MethodGet:
		items, err := a.ProviderSvc.ListCatalog()
		if err != nil {
			writeServiceError(w, err)
			return
		}
		for i := range items {
			maskProviderConfig(&items[i])
		}
		writeJSON(w, http.StatusOK, items)
	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (a *App) handleAdminProviderByID(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlatformAdmin(w, r) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/providers/")
	parts := splitPath(path)
	if len(parts) == 0 || !service.IsValidUUID(parts[0]) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid provider id")
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			item, err := a.ProviderSvc.Get(id)
			if err != nil || item.TenantID != "" {
				writeError(w, http.StatusNotFound, "NOT_FOUND", "provider not found")
				return
			}
			maskProviderConfig(item)
			writeJSON(w, http.StatusOK, item)
		case http.MethodPut:
			var in domain.Provider
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
				return
			}
			in.ID = id
			if err := a.ProviderSvc.UpdateCatalog(&in); err != nil {
				writeServiceError(w, err)
				return
			}
			maskProviderConfig(&in)
			writeJSON(w, http.StatusOK, in)
		default:
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		switch parts[1] {
		case "activate":
			if err := a.ProviderSvc.ActivateCatalog(id); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ACTIVE"})
		case "deactivate":
			if err := a.ProviderSvc.DeactivateCatalog(id); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "INACTIVE"})
		default:
			writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		}
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}

func (a *App) handleMyTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	identity, ok := authn.IdentityFromContext(r.Context())
	if !ok || identity.UserID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	items := make([]domain.Tenant, 0, len(identity.TenantIDs))
	for _, tenantID := range identity.TenantIDs {
		item, err := a.TenantSvc.Get(tenantID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		items = append(items, *item)
	}
	writeJSON(w, http.StatusOK, items)
}

func tenantIDsForUser(user *domain.User) []string {
	if user == nil || user.TenantID == "" {
		return []string{}
	}
	return []string{user.TenantID}
}

func publicUser(user *domain.User, tenantIDs []string) map[string]any {
	if user == nil {
		return nil
	}
	return map[string]any{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"roles":      authn.NormalizeRoles(user.Roles),
		"tenant_ids": tenantIDs,
	}
}

func (a *App) requirePlatformAdmin(w http.ResponseWriter, r *http.Request) bool {
	identity, ok := authn.IdentityFromContext(r.Context())
	if !ok || identity.UserID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return false
	}
	if !identity.HasRole(authn.RolePlatformAdmin) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return false
	}
	return true
}

func (a *App) requireTenantRole(w http.ResponseWriter, r *http.Request, roles ...string) bool {
	identity, ok := authn.IdentityFromContext(r.Context())
	if !ok || identity.UserID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return false
	}
	for _, role := range roles {
		if identity.HasRole(role) {
			return true
		}
	}
	writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
	return false
}

func (a *App) requireTenantAdmin(w http.ResponseWriter, r *http.Request) bool {
	return a.requireTenantRole(w, r, authn.RoleTenantAdmin)
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
	if !a.requireTenantAdmin(w, r) {
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
	case "transactions":
		a.handleTenantTransactions(w, r, tenantID, parts[2:])
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
	filters, err := parseBoletoFilters(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filters.TenantID = tenantID
	adminDashboard, err := a.BoletoSvc.AdminDashboard(filters)
	if err == nil {
		writeJSON(w, http.StatusOK, tenantDashboardResponse(adminDashboard))
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
		"taxa_sucesso":             0.0,
		"taxa_falha":               0.0,
		"ticket_medio":             int64(0),
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
			summary["valor_total_emitido"] = summary["valor_total_emitido"].(int64) + boleto.AmountCents
		case types.StatusExpired:
			summary["boletos_vencidos"] = summary["boletos_vencidos"].(int) + 1
			summary["valor_total_emitido"] = summary["valor_total_emitido"].(int64) + boleto.AmountCents
		case types.StatusCancelled:
			summary["boletos_cancelados"] = summary["boletos_cancelados"].(int) + 1
			summary["valor_total_emitido"] = summary["valor_total_emitido"].(int64) + boleto.AmountCents
		case types.StatusFailed:
			summary["boletos_com_falha"] = summary["boletos_com_falha"].(int) + 1
		}
	}
	total := summary["total_boletos"].(int)
	sucesso := summary["boletos_emitidos"].(int) + summary["boletos_pagos"].(int) + summary["boletos_vencidos"].(int) + summary["boletos_cancelados"].(int)
	completas := sucesso + summary["boletos_com_falha"].(int)
	if completas > 0 {
		summary["taxa_sucesso"] = float64(sucesso) / float64(completas)
		summary["taxa_falha"] = float64(summary["boletos_com_falha"].(int)) / float64(completas)
	}
	if sucesso > 0 {
		summary["ticket_medio"] = summary["valor_total_emitido"].(int64) / int64(sucesso)
	}
	_ = total
	writeJSON(w, http.StatusOK, summary)
}

func (a *App) handleTenantTransactions(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if len(tail) != 0 || r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	filters, err := parseBoletoFilters(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	filters.TenantID = tenantID
	filters.Limit = parseIntQuery(r, "limit", 50)
	filters.Offset = parseIntQuery(r, "offset", 0)
	transactions, err := a.BoletoSvc.ListTransactions(filters)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, transactions)
}

func tenantDashboardResponse(dashboard *domain.AdminDashboard) map[string]any {
	totals := dashboard.Totals
	return map[string]any{
		"total_boletos":            totals.Boletos,
		"boletos_emitidos":         totals.Issued,
		"boletos_em_processamento": totals.Processing,
		"boletos_pagos":            totals.Paid,
		"boletos_vencidos":         totals.Expired,
		"boletos_cancelados":       totals.Cancelled,
		"boletos_com_falha":        totals.Failed,
		"valor_total_emitido":      totals.AmountCents,
		"taxa_sucesso":             totals.SuccessRate,
		"taxa_falha":               totals.FailureRate,
		"ticket_medio":             totals.AverageTicketCents,
		"by_status":                dashboard.ByStatus,
	}
}

func (a *App) handleTenantUsers(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if len(tail) != 0 || r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	if !a.requireTenantAdmin(w, r) {
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
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "provider not found")
			return
		}
		allowed, err := a.ProviderSvc.IsAllowedForTenant(tenantID, id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if !allowed {
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
			if !a.requireTenantAdmin(w, r) {
				return
			}
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
			if !a.requireTenantAdmin(w, r) {
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
			if !a.requireTenantAdmin(w, r) {
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
			if !a.requireTenantAdmin(w, r) {
				return
			}
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
			if !a.requireTenantAdmin(w, r) {
				return
			}
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
		tenantProviderConfig, err := a.ProviderSvc.GetTenantProvider(tenantID, providerID)
		if err != nil {
			return cfg, nil, err
		}
		provider := tenantProviderConfig.Provider
		cfg = types.ProviderConfig{ID: provider.ID, TenantID: tenantID, Name: provider.Name}
		if tenantProviderConfig.TenantProvider.Config != nil {
			cfg.Config = strings.TrimSpace(*tenantProviderConfig.TenantProvider.Config)
		} else if provider.Config != nil {
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

func parseBoletoFilters(r *http.Request) (domain.BoletoFilters, error) {
	query := r.URL.Query()
	var filters domain.BoletoFilters
	if raw := strings.TrimSpace(query.Get("from")); raw != "" {
		value, err := service.NormalizeDueDate(raw)
		if err != nil {
			return filters, service.ErrValidation
		}
		filters.From = &value
	}
	if raw := strings.TrimSpace(query.Get("to")); raw != "" {
		value, err := service.NormalizeDueDate(raw)
		if err != nil {
			return filters, service.ErrValidation
		}
		filters.To = &value
	}
	filters.TenantID = strings.TrimSpace(query.Get("tenant_id"))
	filters.ProviderID = strings.TrimSpace(query.Get("provider_id"))
	filters.Status = strings.ToUpper(strings.TrimSpace(query.Get("status")))
	filters.ExternalID = strings.TrimSpace(query.Get("external_id"))
	filters.OurNumber = strings.TrimSpace(query.Get("our_number"))
	filters.Document = strings.TrimSpace(query.Get("document"))
	filters.Email = strings.TrimSpace(strings.ToLower(query.Get("email")))
	return filters, nil
}

func parseIntQuery(r *http.Request, name string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func maskProviderConfig(provider *domain.Provider) {
	if provider == nil || provider.Config == nil {
		return
	}
	masked := "***"
	provider.Config = &masked
}

func maskTenantProviders(items []domain.TenantProvider) []domain.TenantProvider {
	for i := range items {
		if items[i].Config != nil {
			masked := "***"
			items[i].Config = &masked
		}
	}
	return items
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
				Email         string  `json:"email"`
				CustomerID    *string `json:"customer_id"`
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
				TenantID:       tenantID,
				CustomerID:     in.CustomerID,
				RecipientEmail: in.Email,
				ProviderID:     in.ProviderID,
				AmountCents:    in.AmountCents,
				DueDate:        dueDate,
				Status:         in.Status,
				ExternalID:     in.ExternalID,
				Barcode:        in.Barcode,
				DigitableLine:  in.DigitableLine,
				OurNumber:      in.OurNumber,
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
