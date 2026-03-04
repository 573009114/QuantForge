package api

import (
	"encoding/json"
	"net/http"

	"quantforge/server/internal/middleware"
	"quantforge/server/internal/model"
	"quantforge/server/internal/service"
)

type RiskHandler struct {
	service *service.RiskService
	audit   *service.AuditService
}

func NewRiskHandler(service *service.RiskService, audit *service.AuditService) *RiskHandler {
	return &RiskHandler{service: service, audit: audit}
}

func (h *RiskHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	var req model.UpsertRiskRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	item, err := h.service.UpsertRule(middleware.TenantID(r.Context()), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if h.audit != nil {
		h.audit.Append(middleware.TenantID(r.Context()), middleware.UserID(r.Context()), "RISK_RULE_UPSERT", "risk_rule", "tenant_rule")
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *RiskHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, ok := h.service.GetRule(middleware.TenantID(r.Context()))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "risk rule not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}
