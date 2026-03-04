package api

import (
	"encoding/json"
	"net/http"
	"time"

	"quantforge/server/internal/model"
	"quantforge/server/internal/service"
)

type AuthHandler struct{ service *service.AuthService }

func NewAuthHandler(service *service.AuthService) *AuthHandler { return &AuthHandler{service: service} }

func (h *AuthHandler) IssueToken(w http.ResponseWriter, r *http.Request) {
	var req model.IssueTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	token, err := h.service.IssueToken(req.TenantID, req.Role, req.UserID, 8*time.Hour)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, model.IssueTokenResponse{Token: token})
}
