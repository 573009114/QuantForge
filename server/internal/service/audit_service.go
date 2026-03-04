package service

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"quantforge/server/internal/model"
)

type AuditService struct {
	mu      sync.RWMutex
	store   map[string][]model.AuditLog
	counter atomic.Uint64
}

func NewAuditService() *AuditService {
	return &AuditService{store: map[string][]model.AuditLog{}}
}

func (s *AuditService) Append(tenantID, userID, action, resource, detail string) model.AuditLog {
	item := model.AuditLog{
		ID:        fmt.Sprintf("audit_%d", s.counter.Add(1)),
		TenantID:  tenantID,
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Detail:    detail,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[tenantID] = append(s.store[tenantID], item)
	return item
}

func (s *AuditService) List(tenantID string) []model.AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.store[tenantID]
	out := make([]model.AuditLog, len(items))
	copy(out, items)
	return out
}
