package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"quantforge/server/internal/middleware"
	"quantforge/server/internal/model"
	"quantforge/server/internal/service"
)

type SimHandler struct{ service *service.SimService }

func NewSimHandler(service *service.SimService) *SimHandler { return &SimHandler{service: service} }

func (h *SimHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req model.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	item, err := h.service.CreateAccount(middleware.TenantID(r.Context()), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *SimHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": h.service.ListAccounts(middleware.TenantID(r.Context()))})
}

func (h *SimHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req model.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	item, err := h.service.CreateOrder(middleware.TenantID(r.Context()), req)
	if err != nil {
		if errors.Is(err, service.ErrAccountNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "account not found"})
			return
		}
		if errors.Is(err, service.ErrRiskViolation) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *SimHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": h.service.ListOrders(middleware.TenantID(r.Context()))})
}
