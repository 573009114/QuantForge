package service

import (
	"testing"
	"time"

	"quantforge/server/internal/model"
)

func TestSandboxRunSuccess(t *testing.T) {
	strategySvc := NewStrategyService()
	st, _ := strategySvc.Create("t1", model.CreateStrategyRequest{Name: "s1", Version: "v1"})
	svc := NewSandboxService(strategySvc)
	run, err := svc.CreateRun("t1", model.CreateSandboxRunRequest{StrategyID: st.ID})
	if err != nil {
		t.Fatalf("create run failed: %v", err)
	}
	final := waitRun(t, svc, "t1", run.ID)
	if final.Status != model.SandboxRunSucceeded {
		t.Fatalf("expected SUCCEEDED got %s", final.Status)
	}
}

func TestSandboxRunStop(t *testing.T) {
	strategySvc := NewStrategyService()
	st, _ := strategySvc.Create("t1", model.CreateStrategyRequest{Name: "s1", Version: "v1"})
	svc := NewSandboxService(strategySvc)
	run, _ := svc.CreateRun("t1", model.CreateSandboxRunRequest{StrategyID: st.ID})
	_ = svc.StopRun("t1", run.ID)
	final := waitRun(t, svc, "t1", run.ID)
	if final.Status != model.SandboxRunStopped {
		t.Fatalf("expected STOPPED got %s", final.Status)
	}
}

func waitRun(t *testing.T, svc *SandboxService, tenant, id string) model.SandboxRun {
	t.Helper()
	d := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(d) {
		r, _ := svc.GetRun(tenant, id)
		if r.Status == model.SandboxRunSucceeded || r.Status == model.SandboxRunFailed || r.Status == model.SandboxRunStopped || r.Status == model.SandboxRunTimeout {
			return r
		}
		time.Sleep(5 * time.Millisecond)
	}
	r, _ := svc.GetRun(tenant, id)
	t.Fatalf("run not finished in time, status=%s", r.Status)
	return model.SandboxRun{}
}
