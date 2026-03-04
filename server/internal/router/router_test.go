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
type createAccountResp struct {
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
	var strategy createStrategyResp
	_ = json.NewDecoder(res.Body).Decode(&strategy)
	btBody := []byte(`{"strategyId":"` + strategy.ID + `"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(btBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Viewer")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", res.Code)
	}
}

func TestSimAccountAndOrderEndpoints(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sim/accounts", bytes.NewReader([]byte(`{"balance":10000}`)))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Quant Developer")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}
	var acc createAccountResp
	_ = json.NewDecoder(res.Body).Decode(&acc)

	orderBody := []byte(`{"accountId":"` + acc.ID + `","symbol":"BTCUSDT","side":"BUY","qty":1,"price":1000}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sim/orders", bytes.NewReader(orderBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Quant Developer")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/sim/orders", nil)
	req.Header.Set("X-Tenant-ID", "tenant-a")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
}
