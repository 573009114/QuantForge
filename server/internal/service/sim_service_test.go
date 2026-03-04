package service

import (
	"testing"

	"quantforge/server/internal/model"
)

func TestSimAccountAndOrderFlow(t *testing.T) {
	riskSvc := NewRiskService()
	svc := NewSimService(riskSvc)
	acc, err := svc.CreateAccount("t1", model.CreateAccountRequest{Balance: 10000})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	ord, err := svc.CreateOrder("t1", model.CreateOrderRequest{AccountID: acc.ID, Symbol: "btcusdt", Side: "BUY", Qty: 1, Price: 1000})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if ord.Status != model.OrderStatusFilled {
		t.Fatalf("expected FILLED got %s", ord.Status)
	}
	accounts := svc.ListAccounts("t1")
	if len(accounts) != 1 || accounts[0].Balance != 9000 {
		t.Fatalf("unexpected balance: %+v", accounts)
	}
}

func TestSimTenantIsolation(t *testing.T) {
	riskSvc := NewRiskService()
	svc := NewSimService(riskSvc)
	acc, _ := svc.CreateAccount("t1", model.CreateAccountRequest{Balance: 100})
	if _, err := svc.CreateOrder("t2", model.CreateOrderRequest{AccountID: acc.ID, Symbol: "AAPL", Side: "BUY", Qty: 1, Price: 1}); err == nil {
		t.Fatal("expected account not found for other tenant")
	}
}

func TestSimRiskCheck(t *testing.T) {
	riskSvc := NewRiskService()
	_, _ = riskSvc.UpsertRule("t1", model.UpsertRiskRuleRequest{MaxOrderNotional: 100, MaxDailyLoss: 5000, MaxPositionPercent: 0.5})
	svc := NewSimService(riskSvc)
	acc, _ := svc.CreateAccount("t1", model.CreateAccountRequest{Balance: 1000})
	if _, err := svc.CreateOrder("t1", model.CreateOrderRequest{AccountID: acc.ID, Symbol: "BTCUSDT", Side: "BUY", Qty: 1, Price: 200}); err == nil {
		t.Fatal("expected risk violation")
	}
}
