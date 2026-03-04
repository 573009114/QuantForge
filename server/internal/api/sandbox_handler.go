package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"quantforge/server/internal/middleware"
	"quantforge/server/internal/model"
	"quantforge/server/internal/service"
)

type SandboxHandler struct {
	service *service.SandboxService
	audit   *service.AuditService
}

func NewSandboxHandler(service *service.SandboxService, audit *service.AuditService) *SandboxHandler {
	return &SandboxHandler{service: service, audit: audit}
}

func (h *SandboxHandler) CreateRun(w http.ResponseWriter, r *http.Request) {
	var req model.CreateSandboxRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	item, err := h.service.CreateRun(middleware.TenantID(r.Context()), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if h.audit != nil {
		h.audit.Append(middleware.TenantID(r.Context()), middleware.UserID(r.Context()), "SANDBOX_RUN_CREATE", "sandbox_run", item.ID)
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *SandboxHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": h.service.ListRuns(middleware.TenantID(r.Context()))})
}

func (h *SandboxHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/sandbox/runs/")
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox run not found"})
		return
	}
	item, err := h.service.GetRun(middleware.TenantID(r.Context()), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox run not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *SandboxHandler) StopRun(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/sandbox/runs/")
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox run not found"})
		return
	}
	if err := h.service.StopRun(middleware.TenantID(r.Context()), id); err != nil {
		if errors.Is(err, service.ErrSandboxRunNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox run not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if h.audit != nil {
		h.audit.Append(middleware.TenantID(r.Context()), middleware.UserID(r.Context()), "SANDBOX_RUN_STOP", "sandbox_run", id)
	}
	w.WriteHeader(http.StatusNoContent)
}
