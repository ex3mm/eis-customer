package api

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func authMiddleware(enabled bool, apiKey string, next http.Handler) http.Handler {
	if !enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/docs" || strings.HasPrefix(r.URL.Path, "/docs/") || r.URL.Path == "/openapi.yaml" {
			next.ServeHTTP(w, r)
			return
		}
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		provided := ""
		if strings.HasPrefix(header, prefix) {
			provided = strings.TrimSpace(strings.TrimPrefix(header, prefix))
		}
		if subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
			writeProblem(w, unauthorized(r.URL.Path))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		logger.Info("HTTP request", "method", r.Method, "path", r.URL.Path, "status", wrapped.status, "duration", time.Since(started))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
