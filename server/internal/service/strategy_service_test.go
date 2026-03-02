package service

import (
	"testing"

	"quantforge/server/internal/model"
)

func TestCreateAndGetByTenant(t *testing.T) {
	svc := NewStrategyService()
	created, err := svc.Create("t1", model.CreateStrategyRequest{Name: "mean-revert", Version: "v1"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.TenantID != "t1" {
		t.Fatalf("expected tenant t1 got %s", created.TenantID)
	}

	if _, err = svc.Get("t2", created.ID); err == nil {
		t.Fatal("expected not found on different tenant")
	}

	loaded, err := svc.Get("t1", created.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if loaded.ID != created.ID {
		t.Fatalf("expected same id")
	}
}
