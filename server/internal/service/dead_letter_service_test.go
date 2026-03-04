package service

import "testing"

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
