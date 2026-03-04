package service

import "testing"

func TestAuditAppendAndList(t *testing.T) {
	svc := NewAuditService()
	svc.Append("t1", "u1", "ACT", "res", "d1")
	svc.Append("t1", "u2", "ACT2", "res", "d2")
	items := svc.List("t1")
	if len(items) != 2 {
		t.Fatalf("expected 2 items got %d", len(items))
	}
	if items[0].ID == items[1].ID {
		t.Fatal("expected distinct ids")
	}
}
