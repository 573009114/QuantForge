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

type StrategyHandler struct {
	service *service.StrategyService
}

func NewStrategyHandler(service *service.StrategyService) *StrategyHandler {
	return &StrategyHandler{service: service}
}

func (h *StrategyHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantID(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"items": h.service.List(tenantID)})
}

func (h *StrategyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	created, err := h.service.Create(middleware.TenantID(r.Context()), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *StrategyHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := strategyIDFromPath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "strategy not found"})
		return
	}
	item, err := h.service.Get(middleware.TenantID(r.Context()), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "strategy not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *StrategyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := strategyIDFromPath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "strategy not found"})
		return
	}
	var req model.UpdateStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	updated, err := h.service.Update(middleware.TenantID(r.Context()), id, req)
	if err != nil {
		if errors.Is(err, service.ErrStrategyNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "strategy not found"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *StrategyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := strategyIDFromPath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "strategy not found"})
		return
	}
	if err := h.service.Delete(middleware.TenantID(r.Context()), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "strategy not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func strategyIDFromPath(path string) (string, bool) {
	id := strings.TrimPrefix(path, "/api/v1/strategies/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
