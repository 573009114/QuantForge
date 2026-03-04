package service

import (
	"testing"
	"time"

	"quantforge/server/internal/model"
)

func TestBacktestTriggerAndTenantIsolation(t *testing.T) {
	strategySvc := NewStrategyService()
	created, err := strategySvc.Create("t1", model.CreateStrategyRequest{Name: "s1", Version: "v1"})
	if err != nil {
		t.Fatalf("strategy create failed: %v", err)
	}

	btSvc := NewBacktestService(strategySvc)
	job, err := btSvc.Trigger("t1", model.CreateBacktestRequest{StrategyID: created.ID, Priority: 8})
	if err != nil {
		t.Fatalf("trigger failed: %v", err)
	}
	if job.Status != model.BacktestStatusQueued {
		t.Fatalf("expected QUEUED got %s", job.Status)
	}

	final := waitBacktestStatus(t, btSvc, "t1", job.ID)
	if final.Status != model.BacktestStatusDone {
		t.Fatalf("expected DONE got %s", final.Status)
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
	if _, err := btSvc.Trigger("t1", model.CreateBacktestRequest{StrategyID: "stg_999", Priority: 100}); err == nil {
		t.Fatal("expected priority validation error")
	}
}

func TestBacktestRetryAndFail(t *testing.T) {
	strategySvc := NewStrategyService()
	created, _ := strategySvc.Create("t1", model.CreateStrategyRequest{Name: "s2", Version: "v1"})
	btSvc := NewBacktestService(strategySvc)
	job, err := btSvc.Trigger("t1", model.CreateBacktestRequest{
		StrategyID: created.ID,
		MaxRetries: 2,
		Parameters: map[string]interface{}{"forceFail": true},
	})
	if err != nil {
		t.Fatalf("trigger failed: %v", err)
	}
	final := waitBacktestStatus(t, btSvc, "t1", job.ID)
	if final.Status != model.BacktestStatusFailed {
		t.Fatalf("expected FAILED got %s", final.Status)
	}
	if final.RetryCount != 3 {
		t.Fatalf("expected retryCount=3 got %d", final.RetryCount)
	}
}

func waitBacktestStatus(t *testing.T, svc *BacktestService, tenantID, id string) model.BacktestJob {
	t.Helper()
	deadline := time.Now().Add(120 * time.Millisecond)
	for time.Now().Before(deadline) {
		job, err := svc.Get(tenantID, id)
		if err == nil && (job.Status == model.BacktestStatusDone || job.Status == model.BacktestStatusFailed) {
			return job
		}
		time.Sleep(5 * time.Millisecond)
	}
	job, _ := svc.Get(tenantID, id)
	t.Fatalf("backtest job did not finish in time, last status=%s", job.Status)
	return model.BacktestJob{}
}
