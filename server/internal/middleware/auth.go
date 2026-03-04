package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"quantforge/server/internal/service"
)

type contextKey string

const (
	TenantHeader = "X-Tenant-ID"
	RoleHeader   = "X-Role"
	UserHeader   = "X-User-ID"
	TenantKey    = contextKey("tenantID")
	RoleKey      = contextKey("role")
	UserKey      = contextKey("userID")
)

func WithTenantAndRole(authService *service.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, role, userID := fromToken(authService, r.Header.Get("Authorization"))
		if tenantID == "" {
			tenantID = strings.TrimSpace(r.Header.Get(TenantHeader))
			role = strings.TrimSpace(r.Header.Get(RoleHeader))
			userID = strings.TrimSpace(r.Header.Get(UserHeader))
		}
		if tenantID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant identity"})
			return
		}
		if role == "" {
			role = "Viewer"
		}
		if userID == "" {
			userID = "anonymous"
		}

		ctx := context.WithValue(r.Context(), TenantKey, tenantID)
		ctx = context.WithValue(ctx, RoleKey, role)
		ctx = context.WithValue(ctx, UserKey, userID)
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
func Role(ctx context.Context) string   { role, _ := ctx.Value(RoleKey).(string); return role }
func UserID(ctx context.Context) string { user, _ := ctx.Value(UserKey).(string); return user }

func fromToken(authService *service.AuthService, authz string) (tenantID, role, userID string) {
	if authService == nil {
		return "", "", ""
	}
	authz = strings.TrimSpace(authz)
	if !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return "", "", ""
	}
	token := strings.TrimSpace(authz[7:])
	claims, err := authService.ParseToken(token)
	if err != nil {
		return "", "", ""
	}
	return claims.TenantID, claims.Role, claims.UserID
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
