package main

import (
	"encoding/json"
	"mime"
	"net/http"
	"strings"
	"time"

	authn "github.com/kaiorocha/middleware-boletos/backend/internal/auth"
	"github.com/kaiorocha/middleware-boletos/backend/internal/domain"
	"github.com/kaiorocha/middleware-boletos/backend/internal/service"
)

func (a *App) handleAdminCampaignDashboard(w http.ResponseWriter, r *http.Request) {
	if !a.requirePlatformAdmin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, 405, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	var from, to *time.Time
	if raw := r.URL.Query().Get("from"); raw != "" {
		v, e := service.NormalizeDueDate(raw)
		if e != nil {
			writeServiceError(w, e)
			return
		}
		from = &v
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		v, e := service.NormalizeDueDate(raw)
		if e != nil {
			writeServiceError(w, e)
			return
		}
		to = &v
	}
	d, err := a.CampaignSvc.Dashboard(r.URL.Query().Get("tenant_id"), r.URL.Query().Get("campaign_id"), r.URL.Query().Get("provider_id"), r.URL.Query().Get("status"), from, to)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, 200, d)
}

func campaignIdentity(r *http.Request) (string, bool, bool) {
	i, ok := authn.IdentityFromContext(r.Context())
	if !ok {
		return "", false, false
	}
	return i.UserID, i.HasRole(authn.RolePlatformAdmin), i.HasRole(authn.RoleTenantAdmin)
}
func (a *App) requireCampaignMutation(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, platform, admin := campaignIdentity(r)
	if id == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return "", false
	}
	if !platform && !admin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return "", false
	}
	return id, true
}

func (a *App) handleTenantCampaigns(w http.ResponseWriter, r *http.Request, tenantID string, tail []string) {
	if a.CampaignSvc == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "campaign service not configured")
		return
	}
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodPost:
			userID, ok := a.requireCampaignMutation(w, r)
			if !ok {
				return
			}
			var in domain.Campaign
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				writeError(w, 400, "VALIDATION_ERROR", "invalid payload")
				return
			}
			in.TenantID = tenantID
			if err := a.CampaignSvc.Create(&in, userID, requestID(r)); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, 201, in)
		case http.MethodGet:
			f := domain.CampaignFilters{TenantID: tenantID, Status: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status"))), Search: strings.TrimSpace(r.URL.Query().Get("q")), Limit: parseIntQuery(r, "limit", 50), Offset: parseIntQuery(r, "offset", 0)}
			if raw := r.URL.Query().Get("from"); raw != "" {
				v, e := service.NormalizeDueDate(raw)
				if e != nil {
					writeServiceError(w, e)
					return
				}
				f.From = &v
			}
			if raw := r.URL.Query().Get("to"); raw != "" {
				v, e := service.NormalizeDueDate(raw)
				if e != nil {
					writeServiceError(w, e)
					return
				}
				f.To = &v
			}
			items, total, err := a.CampaignSvc.List(f)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"items": items, "total": total, "limit": f.Limit, "offset": f.Offset})
		default:
			writeError(w, 405, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}
	campaign, err := a.CampaignSvc.Get(tenantID, tail[0])
	if err != nil {
		writeError(w, 404, "NOT_FOUND", "campaign not found")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, 200, campaign)
		case http.MethodPut:
			userID, ok := a.requireCampaignMutation(w, r)
			if !ok {
				return
			}
			var in domain.Campaign
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				writeError(w, 400, "VALIDATION_ERROR", "invalid payload")
				return
			}
			campaign.Name, campaign.Description = in.Name, in.Description
			if err := a.CampaignSvc.Update(campaign, userID, requestID(r)); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, 200, campaign)
		default:
			writeError(w, 405, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}
	if len(tail) == 3 && tail[1] == "imports" && tail[2] == "preview" && r.Method == http.MethodPost {
		userID, ok := a.requireCampaignMutation(w, r)
		if !ok {
			return
		}
		media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if media != "text/csv" && media != "application/csv" && media != "application/vnd.ms-excel" {
			writeError(w, 415, "INVALID_FILE_TYPE", "Envie um arquivo CSV.")
			return
		}
		preview, err := a.CampaignSvc.PreviewCSV(r.Body, campaign, userID, requestID(r))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, 200, preview)
		return
	}
	if len(tail) == 4 && tail[1] == "imports" && tail[3] == "confirm" && r.Method == http.MethodPost {
		userID, ok := a.requireCampaignMutation(w, r)
		if !ok {
			return
		}
		providers, err := a.ProviderSvc.ListByTenant(tenantID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		providerID := ""
		for _, p := range providers {
			if p.Status == "ACTIVE" {
				providerID = p.ID
				break
			}
		}
		if providerID == "" {
			writeError(w, 403, "PROVIDER_NOT_ALLOWED", "tenant has no active provider")
			return
		}
		n, err := a.CampaignSvc.Confirm(campaign, tail[2], providerID, userID, requestID(r))
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, 201, map[string]any{"imported": n, "status": domain.CampaignReady})
		return
	}
	if len(tail) == 2 && tail[1] == "issuance" && r.Method == http.MethodPost {
		userID, ok := a.requireCampaignMutation(w, r)
		if !ok {
			return
		}
		if err := a.CampaignSvc.Start(campaign, userID, requestID(r)); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, 202, map[string]string{"status": domain.CampaignProcessing})
		return
	}
	if len(tail) == 2 && tail[1] == "analytics" && r.Method == http.MethodGet {
		m, err := a.CampaignSvc.Metrics(tenantID, campaign.ID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, 200, m)
		return
	}
	writeError(w, 404, "NOT_FOUND", "route not found")
}
