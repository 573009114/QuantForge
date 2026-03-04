package model

import "time"

type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "CREATED"
	OrderStatusSubmitted OrderStatus = "SUBMITTED"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

type SimAccount struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	Balance   float64   `json:"balance"`
	Equity    float64   `json:"equity"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SimOrder struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenantId"`
	AccountID   string      `json:"accountId"`
	Symbol      string      `json:"symbol"`
	Side        string      `json:"side"`
	Qty         float64     `json:"qty"`
	Price       float64     `json:"price"`
	Status      OrderStatus `json:"status"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	SubmittedAt *time.Time  `json:"submittedAt,omitempty"`
	FilledAt    *time.Time  `json:"filledAt,omitempty"`
}

type CreateAccountRequest struct {
	Balance float64 `json:"balance"`
}

type CreateOrderRequest struct {
	AccountID string  `json:"accountId"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Qty       float64 `json:"qty"`
	Price     float64 `json:"price"`
}
