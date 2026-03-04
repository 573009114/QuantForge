package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Notifier interface {
	Notify(event string, payload map[string]any)
}

type NoopNotifier struct{}

func (NoopNotifier) Notify(string, map[string]any) {}

type WebhookNotifier struct {
	url    string
	client *http.Client
}

func NewNotifierFromEnv() Notifier {
	url := strings.TrimSpace(os.Getenv("QF_ALERT_WEBHOOK_URL"))
	if url == "" {
		return NoopNotifier{}
	}
	return &WebhookNotifier{url: url, client: &http.Client{Timeout: 2 * time.Second}}
}

func (n *WebhookNotifier) Notify(event string, payload map[string]any) {
	body, err := json.Marshal(map[string]any{"event": event, "payload": payload, "ts": time.Now().UTC().Format(time.RFC3339)})
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil || resp == nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}
