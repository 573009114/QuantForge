package api

import (
	"net/http"

	"quantforge/server/internal/middleware"
	"quantforge/server/internal/service"
)

type AuditHandler struct{ service *service.AuditService }

func NewAuditHandler(service *service.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": h.service.List(middleware.TenantID(r.Context()))})
}
