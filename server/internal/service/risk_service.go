package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"quantforge/server/internal/model"
	"quantforge/server/internal/store"
)

var ErrRiskViolation = errors.New("risk violation")

type RiskService struct {
	mu    sync.RWMutex
	rules map[string]model.RiskRule
	pg    *store.PostgresStore
}

func NewRiskService() *RiskService {
	return &RiskService{rules: map[string]model.RiskRule{}}
}

func (s *RiskService) WithPostgres(pg *store.PostgresStore) *RiskService {
	s.pg = pg
	return s
}

func (s *RiskService) UpsertRule(tenantID string, req model.UpsertRiskRuleRequest) (model.RiskRule, error) {
	if req.MaxOrderNotional <= 0 || req.MaxDailyLoss <= 0 || req.MaxPositionPercent <= 0 || req.MaxPositionPercent > 1 {
		return model.RiskRule{}, errors.New("invalid risk rule values")
	}
	rule := model.RiskRule{
		TenantID:           tenantID,
		MaxOrderNotional:   req.MaxOrderNotional,
		MaxDailyLoss:       req.MaxDailyLoss,
		MaxPositionPercent: req.MaxPositionPercent,
		UpdatedAt:          time.Now(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[tenantID] = rule
	if s.pg != nil {
		_ = s.pg.SaveRiskRule(rule)
	}
	return rule, nil
}

func (s *RiskService) GetRule(tenantID string) (model.RiskRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rules[tenantID]
	if ok {
		return r, true
	}
	if s.pg != nil {
		if it, found, err := s.pg.LoadRiskRule(tenantID); err == nil && found {
			return it, true
		}
	}
	return r, ok
}

func (s *RiskService) ValidateOrder(tenantID string, account model.SimAccount, orderNotional float64) error {
	rule, ok := s.GetRule(tenantID)
	if !ok {
		return nil // no rule means pass in bootstrap stage
	}
	if orderNotional > rule.MaxOrderNotional {
		return fmt.Errorf("%w: order notional exceeds maxOrderNotional", ErrRiskViolation)
	}
	if orderNotional > account.Equity*rule.MaxPositionPercent {
		return fmt.Errorf("%w: order notional exceeds maxPositionPercent", ErrRiskViolation)
	}
	return nil
}
