package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestForceHTTPSRedirect(t *testing.T) {
	_ = os.Setenv("QF_FORCE_HTTPS", "true")
	defer os.Unsetenv("QF_FORCE_HTTPS")

	h := ForceHTTPS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "http://example.com/health", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusPermanentRedirect {
		t.Fatalf("expected 308 got %d", res.Code)
	}
}

func TestForceHTTPSAllowForwardedHTTPS(t *testing.T) {
	_ = os.Setenv("QF_FORCE_HTTPS", "true")
	defer os.Unsetenv("QF_FORCE_HTTPS")

	h := ForceHTTPS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "http://example.com/health", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
}
