package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"quantforge/server/internal/model"
)

var ErrSandboxRunNotFound = errors.New("sandbox run not found")

type SandboxService struct {
	mu              sync.RWMutex
	store           map[string]map[string]model.SandboxRun
	stops           map[string]chan struct{}
	counter         atomic.Uint64
	strategyService *StrategyService
}

func NewSandboxService(strategyService *StrategyService) *SandboxService {
	return &SandboxService{store: map[string]map[string]model.SandboxRun{}, stops: map[string]chan struct{}{}, strategyService: strategyService}
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
	stop := make(chan struct{}, 1)
	s.stops[tenantID+":"+id] = stop
	s.mu.Unlock()

	go s.execute(tenantID, id, req.ForceFail, stop)
	return item, nil
}

func (s *SandboxService) execute(tenantID, id string, forceFail bool, stop chan struct{}) {
	s.mu.Lock()
	item := s.store[tenantID][id]
	now := time.Now()
	item.Status = model.SandboxRunRunning
	item.StartedAt = &now
	item.UpdatedAt = now
	s.store[tenantID][id] = item
	s.mu.Unlock()

	runDuration := 20 * time.Millisecond
	timeout := time.Duration(item.Policy.TimeoutSec) * time.Second
	select {
	case <-time.After(runDuration):
		s.finishRun(tenantID, id, forceFail)
	case <-time.After(timeout):
		s.finishWithStatus(tenantID, id, model.SandboxRunTimeout, "execution timeout")
	case <-stop:
		s.finishWithStatus(tenantID, id, model.SandboxRunStopped, "stopped by user")
	}
}

func (s *SandboxService) finishRun(tenantID, id string, forceFail bool) {
	if forceFail {
		s.finishWithStatus(tenantID, id, model.SandboxRunFailed, "simulated strategy runtime error")
		return
	}
	s.finishWithStatus(tenantID, id, model.SandboxRunSucceeded, "finished successfully")
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
}

func (s *SandboxService) StopRun(tenantID, id string) error {
	s.mu.RLock()
	_, ok := s.store[tenantID][id]
	stop := s.stops[tenantID+":"+id]
	s.mu.RUnlock()
	if !ok {
		return ErrSandboxRunNotFound
	}
	if stop != nil {
		select {
		case stop <- struct{}{}:
		default:
		}
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
	return base
}
