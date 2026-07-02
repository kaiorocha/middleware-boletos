package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/repository"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

type App struct{
	TenantSvc *service.TenantService
	BoletoSvc *service.BoletoService
	CustRepo  *repository.CustomerRepo
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"data": v})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request){
		writeJSON(w, http.StatusOK, map[string]string{"status":"ok"})
	})

	mux.HandleFunc("/api/v1/tenants", func(w http.ResponseWriter, r *http.Request){
		if r.Method == http.MethodPost {
			var t domain.Tenant
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil { writeError(w, http.StatusBadRequest, "PARSE_ERROR", "invalid payload"); return }
			if err := a.TenantSvc.Create(&t); err != nil { writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error()); return }
			writeJSON(w, http.StatusCreated, t)
			return
		}
		// list
		if r.Method == http.MethodGet {
			out, err := a.TenantSvc.List(); if err != nil { writeError(w, http.StatusInternalServerError, "SERVER_ERROR", err.Error()); return }
			writeJSON(w, http.StatusOK, out)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/v1/tenants/", func(w http.ResponseWriter, r *http.Request){
		// expect /api/v1/tenants/:id
		id := r.URL.Path[len("/api/v1/tenants/"):]
		if id == "" { writeError(w, http.StatusBadRequest, "INVALID_ID", "missing id"); return }
		if r.Method == http.MethodGet {
			t, err := a.TenantSvc.Get(id); if err != nil { writeError(w, http.StatusNotFound, "NOT_FOUND", "tenant not found"); return }
			writeJSON(w, http.StatusOK, t); return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// Customers: POST /api/v1/tenants/:tenantId/customers and GET list
	mux.HandleFunc("/api/v1/tenants/", func(w http.ResponseWriter, r *http.Request){
		// catch-all minimal router for customers/providers/boletos
		path := r.URL.Path
		// customers
		// /api/v1/tenants/:tenantId/customers
		if r.Method == http.MethodPost && len(path) > 20 && path[len(path)-9:] == "/customers" {
			// extract tenantId
			parts := splitPath(path)
			if len(parts) >= 5 {
				tenantId := parts[4]
				var c domain.Customer
				if err := json.NewDecoder(r.Body).Decode(&c); err != nil { writeError(w, http.StatusBadRequest, "PARSE_ERROR", "invalid payload"); return }
				c.TenantID = tenantId
				if err := a.CustRepo.Create(&c); err != nil { writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error()); return }
				writeJSON(w, http.StatusCreated, c); return
			}
		}
		// GET list /api/v1/tenants/:tenantId/customers
		if r.Method == http.MethodGet && len(path) > 20 && path[len(path)-9:] == "/customers" {
			parts := splitPath(path)
			if len(parts) >= 5 {
				tenantId := parts[4]
				out, err := a.CustRepo.ListByTenant(tenantId); if err != nil { writeError(w, http.StatusInternalServerError, "SERVER_ERROR", err.Error()); return }
				writeJSON(w, http.StatusOK, out); return
			}
		}
		// boletos
		// POST /api/v1/tenants/:tenantId/boletos
		if r.Method == http.MethodPost && len(path) > 17 && path[len(path)-8:] == "/boletos" {
			parts := splitPath(path)
			if len(parts) >= 5 {
				tenantId := parts[4]
				var b domain.Boleto
				if err := json.NewDecoder(r.Body).Decode(&b); err != nil { writeError(w, http.StatusBadRequest, "PARSE_ERROR", "invalid payload"); return }
				b.TenantID = tenantId
				if err := a.BoletoSvc.Create(&b); err != nil { writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error()); return }
				writeJSON(w, http.StatusCreated, b); return
			}
		}
		// GET list /api/v1/tenants/:tenantId/boletos
		if r.Method == http.MethodGet && len(path) > 17 && path[len(path)-8:] == "/boletos" {
			parts := splitPath(path)
			if len(parts) >= 5 {
				tenantId := parts[4]
				out, err := a.BoletoSvc.ListByTenant(tenantId); if err != nil { writeError(w, http.StatusInternalServerError, "SERVER_ERROR", err.Error()); return }
				writeJSON(w, http.StatusOK, out); return
			}
		}
		writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	})

	return mux
}

func splitPath(p string) []string {
	// naive split
	out := []string{}
	cur := ""
	for i:=0;i<len(p);i++ {
		if p[i] == '/' {
			if cur != "" { out = append(out, cur); cur = "" }
			continue
		}
		cur += string(p[i])
	}
	if cur != "" { out = append(out, cur) }
	return out
}
