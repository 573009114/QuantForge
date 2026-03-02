package service

import (
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

type BacktestService struct {
	mu              sync.RWMutex
	store           map[string]map[string]model.BacktestJob
	counter         atomic.Uint64
	strategyService *StrategyService
}

func NewBacktestService(strategyService *StrategyService) *BacktestService {
	return &BacktestService{
		store:           make(map[string]map[string]model.BacktestJob),
		strategyService: strategyService,
	}
}

func (s *BacktestService) Trigger(tenantID string, req model.CreateBacktestRequest) (model.BacktestJob, error) {
	strategyID := strings.TrimSpace(req.StrategyID)
	if strategyID == "" {
		return model.BacktestJob{}, errors.New("strategyId is required")
	}
	if _, err := s.strategyService.Get(tenantID, strategyID); err != nil {
		return model.BacktestJob{}, errors.New("strategy not found")
	}

	now := time.Now()
	id := fmt.Sprintf("bt_%d", s.counter.Add(1))
	item := model.BacktestJob{
		ID:         id,
		TenantID:   tenantID,
		StrategyID: strategyID,
		Status:     model.BacktestStatusQueued,
		Parameters: req.Parameters,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.store[tenantID]; !ok {
		s.store[tenantID] = make(map[string]model.BacktestJob)
	}
	s.store[tenantID][id] = item
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
