package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"quantforge/server/internal/middleware"
	"quantforge/server/internal/model"
	"quantforge/server/internal/service"
)

type BacktestHandler struct {
	service *service.BacktestService
	audit   *service.AuditService
}

func NewBacktestHandler(service *service.BacktestService, audit *service.AuditService) *BacktestHandler {
	return &BacktestHandler{service: service, audit: audit}
}

func (h *BacktestHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	var req model.CreateBacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	item, err := h.service.Trigger(middleware.TenantID(r.Context()), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if h.audit != nil {
		h.audit.Append(middleware.TenantID(r.Context()), middleware.UserID(r.Context()), "BACKTEST_TRIGGER", "backtest", item.ID)
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *BacktestHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": h.service.List(middleware.TenantID(r.Context()))})
}

func (h *BacktestHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/backtests/")
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "backtest not found"})
		return
	}
	item, err := h.service.Get(middleware.TenantID(r.Context()), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "backtest not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}
