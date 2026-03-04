package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	limiter := NewTenantRateLimiter(1, time.Minute)
	h := WithTenantAndRole(nil, limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d", res.Code)
	}
}
