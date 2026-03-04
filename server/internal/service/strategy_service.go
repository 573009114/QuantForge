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
	"quantforge/server/internal/store"
)

var ErrStrategyNotFound = errors.New("strategy not found")

type StrategyService struct {
	mu      sync.RWMutex
	store   map[string]map[string]model.Strategy
	counter atomic.Uint64
	pg      *store.PostgresStore
}

func NewStrategyService() *StrategyService {
	return &StrategyService{store: make(map[string]map[string]model.Strategy)}
}

func (s *StrategyService) WithPostgres(pg *store.PostgresStore) *StrategyService {
	s.pg = pg
	return s
}

func (s *StrategyService) Create(tenantID string, req model.CreateStrategyRequest) (model.Strategy, error) {
	if err := validateCreate(req); err != nil {
		return model.Strategy{}, err
	}
	now := time.Now()
	id := fmt.Sprintf("stg_%d", s.counter.Add(1))
	item := model.Strategy{
		ID:          id,
		TenantID:    tenantID,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Version:     strings.TrimSpace(req.Version),
		Status:      model.StrategyStatusDraft,
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
	if s.pg != nil {
		_ = s.pg.SaveStrategy(item)
	}
	return item, nil
}

func (s *StrategyService) List(tenantID string) []model.Strategy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	strategies := make([]model.Strategy, 0, len(s.store[tenantID]))
	for _, st := range s.store[tenantID] {
		strategies = append(strategies, st)
	}
	if s.pg != nil {
		if items, err := s.pg.LoadStrategies(tenantID); err == nil {
			return items
		}
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

func (s *StrategyService) Update(tenantID, strategyID string, req model.UpdateStrategyRequest) (model.Strategy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.store[tenantID][strategyID]
	if !ok {
		return model.Strategy{}, ErrStrategyNotFound
	}

	if req.Description != "" {
		item.Description = strings.TrimSpace(req.Description)
	}
	if req.Version != "" {
		item.Version = strings.TrimSpace(req.Version)
	}
	if req.Status != "" {
		if req.Status != model.StrategyStatusDraft && req.Status != model.StrategyStatusActive {
			return model.Strategy{}, errors.New("invalid strategy status")
		}
		item.Status = req.Status
	}
	if req.Parameters != nil {
		item.Parameters = req.Parameters
	}
	item.UpdatedAt = time.Now()
	s.store[tenantID][strategyID] = item
	if s.pg != nil {
		_ = s.pg.SaveStrategy(item)
	}
	return item, nil
}

func (s *StrategyService) Delete(tenantID, strategyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.store[tenantID][strategyID]; !ok {
		return ErrStrategyNotFound
	}
	delete(s.store[tenantID], strategyID)
	if s.pg != nil {
		_ = s.pg.DeleteStrategy(tenantID, strategyID)
	}
	return nil
}

func validateCreate(req model.CreateStrategyRequest) error {
	if len(strings.TrimSpace(req.Name)) < 2 {
		return errors.New("name must be at least 2 characters")
	}
	if strings.TrimSpace(req.Version) == "" {
		return errors.New("version is required")
	}
	return nil
}
