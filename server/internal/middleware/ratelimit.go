package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type limiterState struct {
	windowStart time.Time
	count       int
}

type TenantRateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	maxReqs int
	state   map[string]limiterState
}

func NewTenantRateLimiter(maxReqs int, window time.Duration) *TenantRateLimiter {
	if maxReqs <= 0 {
		maxReqs = 200
	}
	if window <= 0 {
		window = time.Second
	}
	return &TenantRateLimiter{window: window, maxReqs: maxReqs, state: map[string]limiterState{}}
}

func (l *TenantRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := TenantID(r.Context())
		if tenantID == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !l.allow(tenantID) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *TenantRateLimiter) allow(tenantID string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	st, ok := l.state[tenantID]
	if !ok || now.Sub(st.windowStart) >= l.window {
		l.state[tenantID] = limiterState{windowStart: now, count: 1}
		return true
	}
	if st.count >= l.maxReqs {
		return false
	}
	st.count++
	l.state[tenantID] = st
	return true
}
