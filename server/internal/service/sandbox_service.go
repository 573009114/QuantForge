package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"quantforge/server/internal/model"
)

var ErrSandboxRunNotFound = errors.New("sandbox run not found")

type sandboxExecutor interface {
	Run(ctx context.Context, strategyID string, policy model.SandboxPolicy, forceFail bool) (string, error)
}

type simulateExecutor struct{}

func (simulateExecutor) Run(ctx context.Context, _ string, _ model.SandboxPolicy, forceFail bool) (string, error) {
	select {
	case <-ctx.Done():
		return "execution timeout", ctx.Err()
	case <-time.After(20 * time.Millisecond):
	}
	if forceFail {
		return "simulated strategy runtime error", errors.New("simulated failure")
	}
	return "finished successfully", nil
}

type dockerExecutor struct{ image string }

func (d dockerExecutor) Run(ctx context.Context, _ string, policy model.SandboxPolicy, forceFail bool) (string, error) {
	if forceFail {
		return "simulated strategy runtime error", errors.New("simulated failure")
	}
	args := []string{"run", "--rm", "--memory", fmt.Sprintf("%dm", policy.MemoryMB), "--cpus", fmt.Sprintf("%.3f", float64(policy.CPUCoresMilli)/1000.0)}
	if policy.NetworkNone {
		args = append(args, "--network=none")
	}
	if policy.ReadOnlyFS {
		args = append(args, "--read-only")
	}
	args = append(args, d.image)
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		msg = "docker run completed"
	}
	return msg, err
}

type SandboxService struct {
	mu              sync.RWMutex
	store           map[string]map[string]model.SandboxRun
	cancels         map[string]context.CancelFunc
	counter         atomic.Uint64
	strategyService *StrategyService
	executor        sandboxExecutor
}

func NewSandboxService(strategyService *StrategyService) *SandboxService {
	execMode := strings.ToLower(strings.TrimSpace(os.Getenv("QF_SANDBOX_EXECUTOR")))
	var executor sandboxExecutor = simulateExecutor{}
	if execMode == "docker" {
		image := strings.TrimSpace(os.Getenv("QF_SANDBOX_IMAGE"))
		if image == "" {
			image = "strategy_runner"
		}
		executor = dockerExecutor{image: image}
	}
	return &SandboxService{store: map[string]map[string]model.SandboxRun{}, cancels: map[string]context.CancelFunc{}, strategyService: strategyService, executor: executor}
}

func (s *SandboxService) CreateRun(tenantID string, req model.CreateSandboxRunRequest) (model.SandboxRun, error) {
	strategyID := strings.TrimSpace(req.StrategyID)
	if strategyID == "" {
		return model.SandboxRun{}, errors.New("strategyId is required")
	}
	if _, err := s.strategyService.Get(tenantID, strategyID); err != nil {
		return model.SandboxRun{}, errors.New("strategy not found")
	}
	policy := defaultPolicy()
	if req.Policy != nil {
		policy = mergePolicy(policy, *req.Policy)
	}
	if policy.TimeoutSec <= 0 || policy.TimeoutSec > 120 {
		return model.SandboxRun{}, errors.New("timeoutSec must be between 1 and 120")
	}

	now := time.Now()
	id := fmt.Sprintf("run_%d", s.counter.Add(1))
	item := model.SandboxRun{ID: id, TenantID: tenantID, StrategyID: strategyID, Status: model.SandboxRunQueued, Policy: policy, CreatedAt: now, UpdatedAt: now}

	s.mu.Lock()
	if _, ok := s.store[tenantID]; !ok {
		s.store[tenantID] = map[string]model.SandboxRun{}
	}
	s.store[tenantID][id] = item
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(policy.TimeoutSec)*time.Second)
	s.mu.Lock()
	s.cancels[tenantID+":"+id] = cancel
	s.mu.Unlock()
	go s.execute(ctx, tenantID, id, req.ForceFail)
	return item, nil
}

func (s *SandboxService) execute(ctx context.Context, tenantID, id string, forceFail bool) {
	s.mu.Lock()
	item := s.store[tenantID][id]
	now := time.Now()
	item.Status = model.SandboxRunRunning
	item.StartedAt = &now
	item.UpdatedAt = now
	s.store[tenantID][id] = item
	s.mu.Unlock()

	msg, err := s.executor.Run(ctx, item.StrategyID, item.Policy, forceFail)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		s.finishWithStatus(tenantID, id, model.SandboxRunTimeout, "execution timeout")
		return
	}
	if errors.Is(err, context.Canceled) {
		s.finishWithStatus(tenantID, id, model.SandboxRunStopped, "stopped by user")
		return
	}
	if err != nil {
		s.finishWithStatus(tenantID, id, model.SandboxRunFailed, msg)
		return
	}
	s.finishWithStatus(tenantID, id, model.SandboxRunSucceeded, msg)
}

func (s *SandboxService) finishWithStatus(tenantID, id string, status model.SandboxRunStatus, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.store[tenantID][id]
	if !ok {
		return
	}
	if item.Status == model.SandboxRunStopped || item.Status == model.SandboxRunSucceeded || item.Status == model.SandboxRunFailed || item.Status == model.SandboxRunTimeout {
		return
	}
	now := time.Now()
	item.Status = status
	item.ExitMessage = msg
	item.UpdatedAt = now
	item.FinishedAt = &now
	s.store[tenantID][id] = item
	if c := s.cancels[tenantID+":"+id]; c != nil {
		delete(s.cancels, tenantID+":"+id)
	}
}

func (s *SandboxService) StopRun(tenantID, id string) error {
	s.mu.RLock()
	_, ok := s.store[tenantID][id]
	cancel := s.cancels[tenantID+":"+id]
	s.mu.RUnlock()
	if !ok {
		return ErrSandboxRunNotFound
	}
	if cancel != nil {
		cancel()
	}
	return nil
}

func (s *SandboxService) GetRun(tenantID, id string) (model.SandboxRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.store[tenantID][id]
	if !ok {
		return model.SandboxRun{}, ErrSandboxRunNotFound
	}
	return item, nil
}

func (s *SandboxService) ListRuns(tenantID string) []model.SandboxRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.SandboxRun, 0, len(s.store[tenantID]))
	for _, it := range s.store[tenantID] {
		items = append(items, it)
	}
	return items
}

func defaultPolicy() model.SandboxPolicy {
	return model.SandboxPolicy{MemoryMB: 512, CPUCoresMilli: 1000, NetworkNone: true, ReadOnlyFS: true, TimeoutSec: 30}
}

func mergePolicy(base, in model.SandboxPolicy) model.SandboxPolicy {
	if in.MemoryMB > 0 {
		base.MemoryMB = in.MemoryMB
	}
	if in.CPUCoresMilli > 0 {
		base.CPUCoresMilli = in.CPUCoresMilli
	}
	if in.TimeoutSec > 0 {
		base.TimeoutSec = in.TimeoutSec
	}
	if in.NetworkNone {
		base.NetworkNone = true
	}
	if in.ReadOnlyFS {
		base.ReadOnlyFS = true
	}
	if v := strings.TrimSpace(os.Getenv("QF_SANDBOX_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			base.TimeoutSec = n
		}
	}
	return base
}
