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
	if loaded.Status != model.StrategyStatusDraft {
		t.Fatalf("expected DRAFT status got %s", loaded.Status)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	svc := NewStrategyService()
	created, err := svc.Create("t1", model.CreateStrategyRequest{Name: "mom", Version: "v1"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := svc.Update("t1", created.ID, model.UpdateStrategyRequest{
		Version: "v2",
		Status:  model.StrategyStatusActive,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Version != "v2" || updated.Status != model.StrategyStatusActive {
		t.Fatalf("unexpected update result: %+v", updated)
	}

	if err = svc.Delete("t1", created.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err = svc.Get("t1", created.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestCreateValidation(t *testing.T) {
	svc := NewStrategyService()
	_, err := svc.Create("t1", model.CreateStrategyRequest{Name: "x", Version: ""})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
