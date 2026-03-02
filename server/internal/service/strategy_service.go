package service

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"quantforge/server/internal/model"
)

var ErrStrategyNotFound = errors.New("strategy not found")

type StrategyService struct {
	mu      sync.RWMutex
	store   map[string]map[string]model.Strategy
	counter atomic.Uint64
}

func NewStrategyService() *StrategyService {
	return &StrategyService{store: make(map[string]map[string]model.Strategy)}
}

func (s *StrategyService) Create(tenantID string, req model.CreateStrategyRequest) (model.Strategy, error) {
	if len(req.Name) < 2 || req.Version == "" {
		return model.Strategy{}, errors.New("name and version are required")
	}
	now := time.Now()
	id := fmt.Sprintf("stg_%d", s.counter.Add(1))
	item := model.Strategy{
		ID:          id,
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		Parameters:  req.Parameters,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.store[tenantID]; !ok {
		s.store[tenantID] = make(map[string]model.Strategy)
	}
	s.store[tenantID][item.ID] = item
	return item, nil
}

func (s *StrategyService) List(tenantID string) []model.Strategy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	strategies := make([]model.Strategy, 0, len(s.store[tenantID]))
	for _, st := range s.store[tenantID] {
		strategies = append(strategies, st)
	}
	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].CreatedAt.After(strategies[j].CreatedAt)
	})
	return strategies
}

func (s *StrategyService) Get(tenantID, strategyID string) (model.Strategy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.store[tenantID][strategyID]
	if !ok {
		return model.Strategy{}, ErrStrategyNotFound
	}
	return item, nil
}
