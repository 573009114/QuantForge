package router

import (
	"encoding/json"
	"net/http"

	"quantforge/server/internal/api"
	"quantforge/server/internal/middleware"
	"quantforge/server/internal/service"
)

func New() http.Handler {
	strategyService := service.NewStrategyService()
	strategyHandler := api.NewStrategyHandler(strategyService)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "quantforge"})
	})

	mux.Handle("/api/v1/strategies", middleware.WithTenantAndRole(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			strategyHandler.List(w, r)
		case http.MethodPost:
			middleware.RequireRole(http.HandlerFunc(strategyHandler.Create), "Admin", "Quant Developer").ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))

	mux.Handle("/api/v1/strategies/", middleware.WithTenantAndRole(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	return mux
}
