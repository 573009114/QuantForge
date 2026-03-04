package service

import (
	"errors"
	"testing"

	"quantforge/server/internal/model"
)

func TestRiskRuleUpsertAndValidate(t *testing.T) {
	svc := NewRiskService()
	_, err := svc.UpsertRule("t1", model.UpsertRiskRuleRequest{MaxOrderNotional: 1000, MaxDailyLoss: 5000, MaxPositionPercent: 0.5})
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	acc := model.SimAccount{ID: "a1", TenantID: "t1", Equity: 2000}
	if err = svc.ValidateOrder("t1", acc, 900); err != nil {
		t.Fatalf("unexpected risk reject: %v", err)
	}
	if err = svc.ValidateOrder("t1", acc, 1200); err == nil {
		t.Fatal("expected risk violation")
	} else if !errors.Is(err, ErrRiskViolation) {
		t.Fatalf("expected ErrRiskViolation got %v", err)
	}
}
