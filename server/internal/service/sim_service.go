package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"quantforge/server/internal/model"
	"quantforge/server/internal/store"
)

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrOrderNotFound   = errors.New("order not found")
)

type SimService struct {
	mu           sync.RWMutex
	accounts     map[string]map[string]model.SimAccount
	orders       map[string]map[string]model.SimOrder
	accCounter   atomic.Uint64
	ordCounter   atomic.Uint64
	riskService  *RiskService
	pg           *store.PostgresStore
	matchingConf MatchingConfig
}

func NewSimService(riskService *RiskService) *SimService {
	return &SimService{accounts: map[string]map[string]model.SimAccount{}, orders: map[string]map[string]model.SimOrder{}, riskService: riskService, matchingConf: DefaultMatchingConfig()}
}

func (s *SimService) WithPostgres(pg *store.PostgresStore) *SimService {
	s.pg = pg
	return s
}

func (s *SimService) CreateAccount(tenantID string, req model.CreateAccountRequest) (model.SimAccount, error) {
	if req.Balance <= 0 {
		return model.SimAccount{}, errors.New("balance must be positive")
	}
	now := time.Now()
	acc := model.SimAccount{ID: fmt.Sprintf("acc_%d", s.accCounter.Add(1)), TenantID: tenantID, Balance: req.Balance, Equity: req.Balance, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.accounts[tenantID]; !ok {
		s.accounts[tenantID] = map[string]model.SimAccount{}
	}
	s.accounts[tenantID][acc.ID] = acc
	if s.pg != nil {
		_ = s.pg.SaveAccount(acc)
	}
	return acc, nil
}

func (s *SimService) ListAccounts(tenantID string) []model.SimAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pg != nil {
		if items, err := s.pg.LoadAccounts(tenantID); err == nil {
			return items
		}
	}
	res := make([]model.SimAccount, 0, len(s.accounts[tenantID]))
	for _, a := range s.accounts[tenantID] {
		res = append(res, a)
	}
	return res
}

func (s *SimService) CreateOrder(tenantID string, req model.CreateOrderRequest) (model.SimOrder, error) {
	if req.AccountID == "" || strings.TrimSpace(req.Symbol) == "" || req.Qty <= 0 || req.Price <= 0 {
		return model.SimOrder{}, errors.New("invalid order request")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[tenantID][req.AccountID]
	if !ok {
		return model.SimOrder{}, ErrAccountNotFound
	}
	if s.riskService != nil {
		if err := s.riskService.ValidateOrder(tenantID, acc, req.Qty*req.Price); err != nil {
			return model.SimOrder{}, err
		}
	}
	now := time.Now()
	ord := model.SimOrder{ID: fmt.Sprintf("ord_%d", s.ordCounter.Add(1)), TenantID: tenantID, AccountID: req.AccountID, Symbol: strings.ToUpper(strings.TrimSpace(req.Symbol)), Side: strings.ToUpper(strings.TrimSpace(req.Side)), Qty: req.Qty, Price: req.Price, Status: model.OrderStatusCreated, CreatedAt: now, UpdatedAt: now}
	if _, ok := s.orders[tenantID]; !ok {
		s.orders[tenantID] = map[string]model.SimOrder{}
	}
	t1 := now
	ord.Status = model.OrderStatusSubmitted
	ord.SubmittedAt = &t1
	acc2, fillPrice, _, err := ExecuteOrder(acc, req, s.matchingConf)
	if err != nil {
		if errors.Is(err, ErrInsufficientBalance) {
			return model.SimOrder{}, ErrInsufficientBalance
		}
		return model.SimOrder{}, err
	}
	t2 := now.Add(10 * time.Millisecond)
	ord.Price = fillPrice
	ord.Status = model.OrderStatusFilled
	ord.FilledAt = &t2
	ord.UpdatedAt = t2
	s.orders[tenantID][ord.ID] = ord
	if s.pg != nil {
		_ = s.pg.SaveOrder(ord)
	}
	acc2.UpdatedAt = t2
	s.accounts[tenantID][acc.ID] = acc2
	if s.pg != nil {
		_ = s.pg.SaveAccount(acc2)
	}
	return ord, nil
}

func (s *SimService) ListOrders(tenantID string) []model.SimOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pg != nil {
		if items, err := s.pg.LoadOrders(tenantID); err == nil {
			return items
		}
	}
	res := make([]model.SimOrder, 0, len(s.orders[tenantID]))
	for _, o := range s.orders[tenantID] {
		res = append(res, o)
	}
	return res
}
