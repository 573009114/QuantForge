package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStrategyRBACAndLifecycle(t *testing.T) {
	h := New()

	createBody := []byte(`{"name":"demo","version":"v1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/strategies", bytes.NewReader(createBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Viewer")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/strategies", bytes.NewReader(createBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Quant Developer")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/strategies", nil)
	req.Header.Set("X-Tenant-ID", "tenant-a")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
}

func TestTenantHeaderRequired(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/strategies", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", res.Code)
	}
}
