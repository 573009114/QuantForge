package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"quantforge/server/internal/model"
)

var ErrBacktestNotFound = errors.New("backtest not found")

type queuedBacktest struct {
	tenantID string
	jobID    string
}

type BacktestService struct {
	mu              sync.RWMutex
	store           map[string]map[string]model.BacktestJob
	queue           BacktestQueue
	counter         atomic.Uint64
	strategyService *StrategyService
}

func NewBacktestService(strategyService *StrategyService) *BacktestService {
	s := &BacktestService{
		store:           make(map[string]map[string]model.BacktestJob),
		strategyService: strategyService,
	}
	if q, ok := newRedisBacktestQueueFromEnv(); ok {
		s.queue = q
	} else {
		s.queue = newMemoryBacktestQueue()
	}
	go s.worker()
	return s
}

func (s *BacktestService) Trigger(tenantID string, req model.CreateBacktestRequest) (model.BacktestJob, error) {
	strategyID := strings.TrimSpace(req.StrategyID)
	if strategyID == "" {
		return model.BacktestJob{}, errors.New("strategyId is required")
	}
	if _, err := s.strategyService.Get(tenantID, strategyID); err != nil {
		return model.BacktestJob{}, errors.New("strategy not found")
	}
	priority := req.Priority
	if priority == 0 {
		priority = 5
	}
	if priority < 1 || priority > 10 {
		return model.BacktestJob{}, errors.New("priority must be between 1 and 10")
	}
	maxRetries := req.MaxRetries
	if maxRetries < 0 || maxRetries > 5 {
		return model.BacktestJob{}, errors.New("maxRetries must be between 0 and 5")
	}

	now := time.Now()
	id := fmt.Sprintf("bt_%d", s.counter.Add(1))
	item := model.BacktestJob{
		ID:         id,
		TenantID:   tenantID,
		StrategyID: strategyID,
		Status:     model.BacktestStatusQueued,
		Priority:   priority,
		MaxRetries: maxRetries,
		Parameters: req.Parameters,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	s.mu.Lock()
	if _, ok := s.store[tenantID]; !ok {
		s.store[tenantID] = make(map[string]model.BacktestJob)
	}
	s.store[tenantID][id] = item
	s.mu.Unlock()
	_ = s.queue.Enqueue(priority, queuedBacktest{tenantID: tenantID, jobID: id})
	return item, nil
}

func (s *BacktestService) List(tenantID string) []model.BacktestJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.BacktestJob, 0, len(s.store[tenantID]))
	for _, it := range s.store[tenantID] {
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].Priority > items[j].Priority
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items
}

func (s *BacktestService) Get(tenantID, id string) (model.BacktestJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.store[tenantID][id]
	if !ok {
		return model.BacktestJob{}, ErrBacktestNotFound
	}
	return item, nil
}

func (s *BacktestService) worker() {
	ctx := context.Background()
	for {
		job, ok, err := s.queue.Dequeue(ctx)
		if err != nil || !ok {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		s.process(job)
	}
}

func (s *BacktestService) process(job queuedBacktest) {
	s.mu.Lock()
	item, ok := s.store[job.tenantID][job.jobID]
	if !ok {
		s.mu.Unlock()
		return
	}
	now := time.Now()
	item.Status = model.BacktestStatusRunning
	item.StartedAt = &now
	item.UpdatedAt = now
	s.store[job.tenantID][job.jobID] = item
	s.mu.Unlock()

	time.Sleep(8 * time.Millisecond)

	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok = s.store[job.tenantID][job.jobID]
	if !ok {
		return
	}

	forceFail, _ := item.Parameters["forceFail"].(bool)
	if forceFail {
		item.RetryCount++
		item.ErrorMessage = "simulated failure"
		item.UpdatedAt = time.Now()
		if item.RetryCount <= item.MaxRetries {
			item.Status = model.BacktestStatusQueued
			s.store[job.tenantID][job.jobID] = item
			_ = s.queue.Enqueue(item.Priority, queuedBacktest{tenantID: job.tenantID, jobID: job.jobID})
			return
		}
		item.Status = model.BacktestStatusFailed
		done := time.Now()
		item.CompletedAt = &done
		s.store[job.tenantID][job.jobID] = item
		return
	}

	item.Status = model.BacktestStatusDone
	item.ErrorMessage = ""
	done := time.Now()
	item.CompletedAt = &done
	item.UpdatedAt = done
	s.store[job.tenantID][job.jobID] = item
}
