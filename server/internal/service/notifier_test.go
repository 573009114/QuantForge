package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWebhookNotifier(t *testing.T) {
	hit := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_ = os.Setenv("QF_ALERT_WEBHOOK_URL", ts.URL)
	defer os.Unsetenv("QF_ALERT_WEBHOOK_URL")

	n := NewNotifierFromEnv()
	n.Notify("TEST", map[string]any{"k": "v"})
	if !hit {
		t.Fatal("expected webhook hit")
	}
}
