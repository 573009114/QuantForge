package service

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"quantforge/server/internal/model"
	"quantforge/server/internal/store"
)

type DeadLetterService struct {
	mu      sync.RWMutex
	store   map[string][]model.DeadLetterJob
	counter atomic.Uint64
	pg      *store.PostgresStore
}

func NewDeadLetterService() *DeadLetterService {
	return &DeadLetterService{store: map[string][]model.DeadLetterJob{}}
}

func (s *DeadLetterService) WithPostgres(pg *store.PostgresStore) *DeadLetterService {
	s.pg = pg
	return s
}

func (s *DeadLetterService) Append(tenantID, jobType, jobID, reason, payload string) model.DeadLetterJob {
	item := model.DeadLetterJob{
		ID:        fmt.Sprintf("dlq_%d", s.counter.Add(1)),
		TenantID:  tenantID,
		JobType:   jobType,
		JobID:     jobID,
		Reason:    reason,
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.store[tenantID] = append(s.store[tenantID], item)
	s.mu.Unlock()
	if s.pg != nil {
		_ = s.pg.AppendDeadLetter(item)
	}
	return item
}

func (s *DeadLetterService) List(tenantID string, limit int) []model.DeadLetterJob {
	if s.pg != nil {
		if items, err := s.pg.LoadDeadLetters(tenantID, limit); err == nil {
			return items
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.store[tenantID]
	if limit <= 0 || limit >= len(items) {
		out := make([]model.DeadLetterJob, len(items))
		copy(out, items)
		return out
	}
	start := len(items) - limit
	out := make([]model.DeadLetterJob, limit)
	copy(out, items[start:])
	return out
}
