package service

import "testing"

type captureNotifier struct {
	event   string
	payload map[string]any
}

func (n *captureNotifier) Notify(event string, payload map[string]any) {
	n.event = event
	n.payload = payload
}

func TestDeadLetterAppendAndList(t *testing.T) {
	svc := NewDeadLetterService()
	svc.Append("t1", "backtest", "bt_1", "failed", "{}")
	items := svc.List("t1", 10)
	if len(items) != 1 {
		t.Fatalf("expected 1 got %d", len(items))
	}
	if items[0].JobID != "bt_1" {
		t.Fatalf("unexpected job id %s", items[0].JobID)
	}
}

func TestDeadLetterAppendNotify(t *testing.T) {
	n := &captureNotifier{}
	svc := NewDeadLetterService().WithNotifier(n)
	svc.Append("t1", "backtest", "bt_2", "failed", "{}")
	if n.event != "DEAD_LETTER_CREATED" {
		t.Fatalf("expected DEAD_LETTER_CREATED got %s", n.event)
	}
	if n.payload["jobId"] != "bt_2" {
		t.Fatalf("expected jobId bt_2 got %v", n.payload["jobId"])
	}
}
