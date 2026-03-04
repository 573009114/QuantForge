package router

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"quantforge/server/internal/api"
	"quantforge/server/internal/middleware"
	"quantforge/server/internal/service"
)

func New() http.Handler {
	authService := service.NewAuthService(os.Getenv("QF_JWT_SECRET"))
	auditService := service.NewAuditService()
	rateLimiter := middleware.NewTenantRateLimiter(300, time.Second)

	authHandler := api.NewAuthHandler(authService)
	auditHandler := api.NewAuditHandler(auditService)
	strategyService := service.NewStrategyService()
	strategyHandler := api.NewStrategyHandler(strategyService, auditService)
	backtestService := service.NewBacktestService(strategyService)
	backtestHandler := api.NewBacktestHandler(backtestService, auditService)
	riskService := service.NewRiskService()
	riskHandler := api.NewRiskHandler(riskService, auditService)
	simService := service.NewSimService(riskService)
	simHandler := api.NewSimHandler(simService, auditService)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "quantforge"})
	})
	mux.HandleFunc("/api/v1/auth/token", authHandler.IssueToken)

	secure := func(h http.Handler) http.Handler {
		return middleware.WithTenantAndRole(authService, rateLimiter.Middleware(h))
	}

	mux.Handle("/api/v1/audit/logs", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		auditHandler.List(w, r)
	})))

	mux.Handle("/api/v1/strategies", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			strategyHandler.List(w, r)
		case http.MethodPost:
			middleware.RequireRole(http.HandlerFunc(strategyHandler.Create), "Admin", "Quant Developer").ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/strategies/", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			strategyHandler.Get(w, r)
		case http.MethodPut:
			middleware.RequireRole(http.HandlerFunc(strategyHandler.Update), "Admin", "Quant Developer").ServeHTTP(w, r)
		case http.MethodDelete:
			middleware.RequireRole(http.HandlerFunc(strategyHandler.Delete), "Admin").ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/backtests", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			backtestHandler.List(w, r)
		case http.MethodPost:
			middleware.RequireRole(http.HandlerFunc(backtestHandler.Trigger), "Admin", "Quant Developer").ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/backtests/", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		backtestHandler.Get(w, r)
	})))

	mux.Handle("/api/v1/risk/rules", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			riskHandler.Get(w, r)
		case http.MethodPut:
			middleware.RequireRole(http.HandlerFunc(riskHandler.Upsert), "Admin", "Risk Manager").ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/sim/accounts", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			simHandler.ListAccounts(w, r)
		case http.MethodPost:
			middleware.RequireRole(http.HandlerFunc(simHandler.CreateAccount), "Admin", "Quant Developer").ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/sim/orders", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			simHandler.ListOrders(w, r)
		case http.MethodPost:
			middleware.RequireRole(http.HandlerFunc(simHandler.CreateOrder), "Admin", "Quant Developer").ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))

	return mux
}
