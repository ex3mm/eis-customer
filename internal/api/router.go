package api

import (
	"log/slog"
	"net/http"

	"eis-customer/internal/api/docs"
	"eis-customer/internal/cache"
	"eis-customer/internal/config"
	"eis-customer/internal/service"
)

func NewRouter(cfg config.Config, service *service.Service, cache *cache.Cache, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []byte(`{"status":"ok"}`))
	})
	if cfg.DocsEnabled {
		docs.Register(mux)
	}
	handler := NewHandler(service, cache, logger)
	mux.HandleFunc("POST /api/v1/accreditation/check", handler.Check)
	return loggingMiddleware(logger, authMiddleware(cfg.AuthEnabled, cfg.APIKey, mux))
}
