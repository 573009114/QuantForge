package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type issueTokenResp struct {
	Token string `json:"token"`
}
type createStrategyResp struct {
	ID string `json:"id"`
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

func TestIssueTokenAndAccessWithBearer(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader([]byte(`{"tenantId":"tenant-a","role":"Quant Developer","userId":"u1"}`)))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
	var tokenResp issueTokenResp
	_ = json.NewDecoder(res.Body).Decode(&tokenResp)

	createBody := []byte(`{"name":"demo","version":"v1"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/strategies", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+tokenResp.Token)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}
}

func TestAuditLogAfterMutation(t *testing.T) {
	h := New()
	createBody := []byte(`{"name":"demo","version":"v1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/strategies", bytes.NewReader(createBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Quant Developer")
	req.Header.Set("X-User-ID", "u2")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs", nil)
	req.Header.Set("X-Tenant-ID", "tenant-a")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
	if !bytes.Contains(res.Body.Bytes(), []byte("STRATEGY_CREATE")) {
		t.Fatalf("expected audit log to contain STRATEGY_CREATE got %s", res.Body.String())
	}
}

func TestSandboxRunLifecycle(t *testing.T) {
	h := New()

	createBody := []byte(`{"name":"demo","version":"v1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/strategies", bytes.NewReader(createBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Quant Developer")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}
	var st createStrategyResp
	_ = json.NewDecoder(res.Body).Decode(&st)

	runBody := []byte(`{"strategyId":"` + st.ID + `"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sandbox/runs", bytes.NewReader(runBody))
	req.Header.Set("X-Tenant-ID", "tenant-a")
	req.Header.Set("X-Role", "Quant Developer")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}
	if !bytes.Contains(res.Body.Bytes(), []byte("QUEUED")) {
		t.Fatalf("expected queued run response")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/sandbox/runs", nil)
	req.Header.Set("X-Tenant-ID", "tenant-a")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
}
