package fetcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"eis-customer/internal/config"
)

func TestFetchRetriesTemporaryErrorOnce(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	f := testFetcher(t, 1)
	body, err := f.Fetch(context.Background(), "fz44", "6317024749", "631701001", server.URL)
	if err != nil || string(body) != "ok" || calls.Load() != 2 {
		t.Fatalf("body=%q calls=%d err=%v", body, calls.Load(), err)
	}
}

func TestFetchDoesNotRetryPermanentError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	f := testFetcher(t, 1)
	_, err := f.Fetch(context.Background(), "fz44", "6317024749", "631701001", server.URL)
	if err == nil || calls.Load() != 1 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func TestFetchWritesAndReadsFixtures(t *testing.T) {
	directory := t.TempDir()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "fixture body")
	}))
	defer server.Close()

	writer := testFetcher(t, 0)
	writer.fixturesMode = "write"
	writer.fixturesDir = directory
	if _, err := writer.Fetch(context.Background(), "fz223", "6317024749", "631701001", server.URL); err != nil {
		t.Fatal(err)
	}

	reader := testFetcher(t, 0)
	reader.fixturesMode = "read"
	reader.fixturesDir = directory
	body, err := reader.Fetch(context.Background(), "fz223", "6317024749", "631701001", "https://unused.invalid")
	if err != nil || string(body) != "fixture body" {
		t.Fatalf("body=%q err=%v", body, err)
	}
}

func testFetcher(t *testing.T, retries int) *Fetcher {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	f, err := New(config.Config{
		RequestTimeout:     time.Second,
		RetryDelay:         0,
		MaxRetries:         retries,
		UserAgent:          "test",
		InsecureSkipVerify: true,
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
