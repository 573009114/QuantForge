package router

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"quantforge/server/internal/api"
	"quantforge/server/internal/middleware"
	"quantforge/server/internal/service"
	"quantforge/server/internal/store"
)

func New() http.Handler {
	authService := service.NewAuthService(os.Getenv("QF_JWT_SECRET"))
	auditService := service.NewAuditService()
	rateLimiter := middleware.NewTenantRateLimiter(300, time.Second)

	var pg *store.PostgresStore
	if os.Getenv("QF_STORAGE") == "postgres" {
		db, err := store.NewPostgresStoreFromEnv()
		if err != nil {
			log.Fatalf("postgres init failed: %v", err)
		}
		if err = db.ApplyMigrationsFromDir("server/migrations"); err != nil {
			if err = db.ApplyMigrationsFromDir("migrations"); err != nil {
				log.Fatalf("postgres migrations failed: %v", err)
			}
		}
		pg = db
	}

	authHandler := api.NewAuthHandler(authService)
	if pg != nil {
		auditService.WithPostgres(pg)
	}
	auditHandler := api.NewAuditHandler(auditService)
	dlqService := service.NewDeadLetterService()
	if pg != nil {
		dlqService.WithPostgres(pg)
	}
	dlqHandler := api.NewDeadLetterHandler(dlqService)
	strategyService := service.NewStrategyService()
	if pg != nil {
		strategyService.WithPostgres(pg)
	}
	strategyHandler := api.NewStrategyHandler(strategyService, auditService)
	backtestService := service.NewBacktestService(strategyService).WithDeadLetter(dlqService)
	backtestHandler := api.NewBacktestHandler(backtestService, auditService)
	riskService := service.NewRiskService()
	if pg != nil {
		riskService.WithPostgres(pg)
	}
	riskHandler := api.NewRiskHandler(riskService, auditService)
	simService := service.NewSimService(riskService)
	if pg != nil {
		simService.WithPostgres(pg)
	}
	simHandler := api.NewSimHandler(simService, auditService)
	sandboxService := service.NewSandboxService(strategyService)
	sandboxHandler := api.NewSandboxHandler(sandboxService, auditService)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "quantforge"})
	})
	mux.HandleFunc("/health/deps", func(w http.ResponseWriter, _ *http.Request) {
		deps := map[string]string{"postgres": "disabled", "redis": "disabled"}
		status := http.StatusOK
		if os.Getenv("QF_STORAGE") == "postgres" {
			deps["postgres"] = "up"
			if pg == nil {
				deps["postgres"] = "down"
				status = http.StatusServiceUnavailable
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				err := pg.Ping(ctx)
				cancel()
				if err != nil {
					deps["postgres"] = "down"
					status = http.StatusServiceUnavailable
				}
			}
		}
		if addr := os.Getenv("QF_REDIS_ADDR"); addr != "" {
			deps["redis"] = "up"
			conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
			if err != nil {
				deps["redis"] = "down"
				status = http.StatusServiceUnavailable
			} else {
				_ = conn.Close()
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": map[bool]string{true: "ok", false: "degraded"}[status == http.StatusOK], "deps": deps})
	})
	mux.HandleFunc("/api/v1/auth/token", authHandler.IssueToken)

	secure := func(h http.Handler) http.Handler {
		return middleware.WithTenantAndRole(authService, rateLimiter.Middleware(h))
	}

	mux.Handle("/api/v1/dead-letter/jobs", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		dlqHandler.List(w, r)
	})))

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

	mux.Handle("/api/v1/sandbox/runs", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			sandboxHandler.ListRuns(w, r)
		case http.MethodPost:
			middleware.RequireRole(http.HandlerFunc(sandboxHandler.CreateRun), "Admin", "Quant Developer").ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/sandbox/runs/", secure(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			sandboxHandler.GetRun(w, r)
		case http.MethodDelete:
			middleware.RequireRole(http.HandlerFunc(sandboxHandler.StopRun), "Admin", "Risk Manager", "Quant Developer").ServeHTTP(w, r)
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
