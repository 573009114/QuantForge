package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const (
	TenantHeader = "X-Tenant-ID"
	RoleHeader   = "X-Role"
	TenantKey    = contextKey("tenantID")
	RoleKey      = contextKey("role")
)

func WithTenantAndRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := strings.TrimSpace(r.Header.Get(TenantHeader))
		if tenantID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant header"})
			return
		}
		role := strings.TrimSpace(r.Header.Get(RoleHeader))
		if role == "" {
			role = "Viewer"
		}
		ctx := context.WithValue(r.Context(), TenantKey, tenantID)
		ctx = context.WithValue(ctx, RoleKey, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(next http.Handler, allowed ...string) http.Handler {
	allow := make(map[string]struct{}, len(allowed))
	for _, role := range allowed {
		allow[role] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(RoleKey).(string)
		if _, ok := allow[role]; !ok {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient role"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func TenantID(ctx context.Context) string {
	tenantID, _ := ctx.Value(TenantKey).(string)
	return tenantID
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
