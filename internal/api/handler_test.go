package api

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"eis-customer/internal/cache"
	"eis-customer/internal/config"
	"eis-customer/internal/service"
	"eis-customer/internal/urlbuilder"
)

type fixtureFetcher struct{ calls atomic.Int32 }

func (f *fixtureFetcher) Fetch(context.Context, string, string, string, string) ([]byte, error) {
	f.calls.Add(1)
	return []byte(`<div class="search-registry-entry-block">ИНН: 6317024749 КПП: 631701001</div>`), nil
}

func TestCheckCachesOnlySuccessfulResponse(t *testing.T) {
	fetcher := &fixtureFetcher{}
	checker := service.New(fetcher, urlbuilder.New("https://example.test/44", nil), urlbuilder.New("https://example.test/223", nil))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(config.Config{}, checker, cache.New(true, time.Minute), logger)
	body := []byte(`{"inn":"6317024749","kpp":"631701001"}`)

	for index, expectedCache := range []string{"MISS", "HIT"} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/accreditation/check", bytes.NewReader(body))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("request %d: status=%d body=%s", index+1, response.Code, response.Body.String())
		}
		if got := response.Header().Get("X-Cache"); got != expectedCache {
			t.Fatalf("request %d: X-Cache=%q", index+1, got)
		}
	}
	if fetcher.calls.Load() != 2 {
		t.Fatalf("upstream calls = %d, want 2", fetcher.calls.Load())
	}
}

func TestCheckRejectsUnknownJSONField(t *testing.T) {
	fetcher := &fixtureFetcher{}
	checker := service.New(fetcher, urlbuilder.New("https://example.test/44", nil), urlbuilder.New("https://example.test/223", nil))
	router := NewRouter(config.Config{}, checker, cache.New(false, time.Minute), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/accreditation/check", bytes.NewBufferString(`{"inn":"6317024749","kpp":"631701001","extra":true}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestBearerAuthorization(t *testing.T) {
	fetcher := &fixtureFetcher{}
	checker := service.New(fetcher, urlbuilder.New("https://example.test/44", nil), urlbuilder.New("https://example.test/223", nil))
	router := NewRouter(
		config.Config{AuthEnabled: true, APIKey: "test-secret"},
		checker,
		cache.New(false, time.Minute),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	body := []byte(`{"inn":"6317024749","kpp":"631701001"}`)

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "wrong token", header: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", header: "Basic test-secret", wantStatus: http.StatusUnauthorized},
		{name: "valid token", header: "Bearer test-secret", wantStatus: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/accreditation/check", bytes.NewReader(body))
			if test.header != "" {
				request.Header.Set("Authorization", test.header)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			if response.Code == http.StatusUnauthorized && response.Header().Get("Content-Type") != "application/problem+json; charset=utf-8" {
				t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestHealthDoesNotRequireAuthorization(t *testing.T) {
	checker := service.New(&fixtureFetcher{}, urlbuilder.New("https://example.test/44", nil), urlbuilder.New("https://example.test/223", nil))
	router := NewRouter(
		config.Config{AuthEnabled: true, APIKey: "test-secret"},
		checker,
		cache.New(false, time.Minute),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
