package service

import (
	"testing"

	"quantforge/server/internal/model"
)

func TestBacktestTriggerAndTenantIsolation(t *testing.T) {
	strategySvc := NewStrategyService()
	created, err := strategySvc.Create("t1", model.CreateStrategyRequest{Name: "s1", Version: "v1"})
	if err != nil {
		t.Fatalf("strategy create failed: %v", err)
	}

	btSvc := NewBacktestService(strategySvc)
	job, err := btSvc.Trigger("t1", model.CreateBacktestRequest{StrategyID: created.ID})
	if err != nil {
		t.Fatalf("trigger failed: %v", err)
	}
	if job.Status != model.BacktestStatusQueued {
		t.Fatalf("expected QUEUED got %s", job.Status)
	}

	if _, err = btSvc.Get("t2", job.ID); err == nil {
		t.Fatal("expected tenant isolation not found")
	}
}

func TestBacktestTriggerValidation(t *testing.T) {
	strategySvc := NewStrategyService()
	btSvc := NewBacktestService(strategySvc)
	if _, err := btSvc.Trigger("t1", model.CreateBacktestRequest{}); err == nil {
		t.Fatal("expected validation error for missing strategyId")
	}
	if _, err := btSvc.Trigger("t1", model.CreateBacktestRequest{StrategyID: "stg_999"}); err == nil {
		t.Fatal("expected strategy not found error")
	}
}
