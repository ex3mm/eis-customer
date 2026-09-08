package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eis-customer/internal/api"
	"eis-customer/internal/cache"
	"eis-customer/internal/config"
	"eis-customer/internal/fetcher"
	"eis-customer/internal/service"
	"eis-customer/internal/urlbuilder"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	httpFetcher, err := fetcher.New(cfg, logger)
	if err != nil {
		logger.Error("fetcher initialization error", "error", err)
		os.Exit(1)
	}
	checker := service.New(
		httpFetcher,
		urlbuilder.New(cfg.URL44, cfg.Params44),
		urlbuilder.New(cfg.URL223, cfg.Params223),
	)
	router := api.NewRouter(cfg, checker, cache.New(cfg.CacheEnabled, cfg.CacheTTL), logger)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      2*cfg.RequestTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server started", "address", server.Addr, "cache_enabled", cfg.CacheEnabled)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdownSignal.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
