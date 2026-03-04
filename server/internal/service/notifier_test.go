package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWebhookNotifier(t *testing.T) {
	type reqBody struct {
		Event string         `json:"event"`
		Data  map[string]any `json:"payload"`
	}
	hit := make(chan reqBody, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected application/json got %s", r.Header.Get("Content-Type"))
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		hit <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	_ = os.Setenv("QF_ALERT_WEBHOOK_URL", ts.URL)
	defer os.Unsetenv("QF_ALERT_WEBHOOK_URL")

	n := NewNotifierFromEnv()
	n.Notify("TEST", map[string]any{"k": "v"})
	got := <-hit
	if got.Event != "TEST" {
		t.Fatalf("expected TEST got %s", got.Event)
	}
	if got.Data["k"] != "v" {
		t.Fatalf("expected payload k=v got %v", got.Data["k"])
	}
}
