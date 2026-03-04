package service

import (
	"errors"
	"math"
	"strings"

	"quantforge/server/internal/model"
)

var ErrInsufficientBalance = errors.New("insufficient balance")

type MatchingConfig struct {
	FeeRate     float64
	SlippageBps float64
}

func DefaultMatchingConfig() MatchingConfig {
	return MatchingConfig{FeeRate: 0.0005, SlippageBps: 2}
}

func ExecuteOrder(acc model.SimAccount, req model.CreateOrderRequest, cfg MatchingConfig) (model.SimAccount, float64, float64, error) {
	side := strings.ToUpper(strings.TrimSpace(req.Side))
	notional := req.Qty * req.Price
	slipFactor := cfg.SlippageBps / 10000.0
	fillPrice := req.Price
	if side == "BUY" {
		fillPrice = req.Price * (1 + slipFactor)
	} else {
		fillPrice = req.Price * (1 - slipFactor)
	}
	fillNotional := req.Qty * fillPrice
	fee := math.Abs(fillNotional * cfg.FeeRate)
	if side == "BUY" {
		cost := fillNotional + fee
		if acc.Balance < cost {
			return acc, fillPrice, fee, ErrInsufficientBalance
		}
		acc.Balance -= cost
	} else {
		acc.Balance += fillNotional - fee
	}
	acc.Equity = acc.Balance
	_ = notional
	return acc, fillPrice, fee, nil
}
