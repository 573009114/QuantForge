package model

import "time"

type RiskRule struct {
	TenantID           string    `json:"tenantId"`
	MaxOrderNotional   float64   `json:"maxOrderNotional"`
	MaxDailyLoss       float64   `json:"maxDailyLoss"`
	MaxPositionPercent float64   `json:"maxPositionPercent"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type UpsertRiskRuleRequest struct {
	MaxOrderNotional   float64 `json:"maxOrderNotional"`
	MaxDailyLoss       float64 `json:"maxDailyLoss"`
	MaxPositionPercent float64 `json:"maxPositionPercent"`
}
