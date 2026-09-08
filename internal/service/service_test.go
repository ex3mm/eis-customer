package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"eis-customer/internal/urlbuilder"
)

type delayedFetcher struct {
	delay time.Duration
	calls atomic.Int32
}

func (f *delayedFetcher) Fetch(context.Context, string, string, string, string) ([]byte, error) {
	f.calls.Add(1)
	time.Sleep(f.delay)
	return []byte(`<div class="search-registry-entry-block">ИНН: 6317024749 КПП: 631701001</div>`), nil
}

func TestCheckFetchesSourcesInParallel(t *testing.T) {
	fetcher := &delayedFetcher{delay: 100 * time.Millisecond}
	service := New(
		fetcher,
		urlbuilder.New("https://example.test/44", nil),
		urlbuilder.New("https://example.test/223", nil),
	)
	started := time.Now()
	response, err := service.Check(context.Background(), "6317024749", "631701001")
	elapsed := time.Since(started)
	if err != nil {
		t.Fatal(err)
	}
	if !response.Found || !response.Sources["fz44"].Found || !response.Sources["fz223"].Found {
		t.Fatalf("unexpected response: %#v", response)
	}
	if fetcher.calls.Load() != 2 {
		t.Fatalf("calls = %d", fetcher.calls.Load())
	}
	if elapsed >= 180*time.Millisecond {
		t.Fatalf("requests appear sequential: %s", elapsed)
	}
}
