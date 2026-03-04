package api

import (
	"net/http"
	"strconv"

	"quantforge/server/internal/middleware"
	"quantforge/server/internal/service"
)

type DeadLetterHandler struct{ service *service.DeadLetterService }

func NewDeadLetterHandler(service *service.DeadLetterService) *DeadLetterHandler {
	return &DeadLetterHandler{service: service}
}

func (h *DeadLetterHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": h.service.List(middleware.TenantID(r.Context()), limit)})
}
