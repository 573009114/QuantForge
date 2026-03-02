package model

import "time"

type BacktestStatus string

const (
	BacktestStatusQueued  BacktestStatus = "QUEUED"
	BacktestStatusRunning BacktestStatus = "RUNNING"
	BacktestStatusDone    BacktestStatus = "DONE"
	BacktestStatusFailed  BacktestStatus = "FAILED"
)

type BacktestJob struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenantId"`
	StrategyID  string                 `json:"strategyId"`
	Status      BacktestStatus         `json:"status"`
	Parameters  map[string]interface{} `json:"parameters"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
}

type CreateBacktestRequest struct {
	StrategyID string                 `json:"strategyId"`
	Parameters map[string]interface{} `json:"parameters"`
}
