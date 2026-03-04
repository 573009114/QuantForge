package service

import (
	"testing"

	"quantforge/server/internal/model"
)

func TestExecuteOrderBuyWithFeeAndSlippage(t *testing.T) {
	acc := model.SimAccount{Balance: 10000, Equity: 10000}
	cfg := MatchingConfig{FeeRate: 0.001, SlippageBps: 10}
	next, fillPrice, fee, err := ExecuteOrder(acc, model.CreateOrderRequest{Side: "BUY", Qty: 1, Price: 1000}, cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if fillPrice <= 1000 || fee <= 0 {
		t.Fatalf("invalid fill/fee: price=%f fee=%f", fillPrice, fee)
	}
	if next.Balance >= acc.Balance {
		t.Fatalf("expected balance drop")
	}
}

func TestExecuteOrderInsufficientBalance(t *testing.T) {
	acc := model.SimAccount{Balance: 10, Equity: 10}
	_, _, _, err := ExecuteOrder(acc, model.CreateOrderRequest{Side: "BUY", Qty: 1, Price: 1000}, DefaultMatchingConfig())
	if err == nil {
		t.Fatal("expected insufficient balance")
	}
}
