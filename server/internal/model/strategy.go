package model

import "time"

type StrategyStatus string

const (
	StrategyStatusDraft  StrategyStatus = "DRAFT"
	StrategyStatusActive StrategyStatus = "ACTIVE"
)

type Strategy struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenantId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Status      StrategyStatus         `json:"status"`
	Parameters  map[string]interface{} `json:"parameters"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

type CreateStrategyRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type UpdateStrategyRequest struct {
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Status      StrategyStatus         `json:"status"`
	Parameters  map[string]interface{} `json:"parameters"`
}
