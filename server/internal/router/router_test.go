package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type createStrategyResp struct {
	ID string `json:"id"`
}

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

func TestBacktestTriggerRBACAndList(t *testing.T) {
	h := New()

	createBody := []byte(`{"name":"alpha","version":"v1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/strategies", bytes.NewReader(createBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Quant Developer")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected strategy create 201 got %d", res.Code)
	}
	var strategy createStrategyResp
	if err := json.NewDecoder(res.Body).Decode(&strategy); err != nil {
		t.Fatalf("decode strategy failed: %v", err)
	}

	btBody := []byte(`{"strategyId":"` + strategy.ID + `"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(btBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Viewer")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(btBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Quant Developer")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/backtests", nil)
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Viewer")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
}
