package config

import (
	"testing"
	"time"
)

func TestDefault223QueryIncludesOnlyRegisteredStatus(t *testing.T) {
	t.Setenv("EIS_223_PARAM_REGISTERED_STATUS", "0")
	t.Setenv("EIS_223_PARAM_REGISTERED_STATUS_ENABLED", "on")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Params223["customer223Status"] != "0" {
		t.Fatalf("status = %q", cfg.Params223["customer223Status"])
	}
	if _, exists := cfg.Params223["customer223Status_1"]; exists {
		t.Fatal("excluded status must not be sent")
	}
}

func TestAuthRequiresAPIKey(t *testing.T) {
	cfg := Config{
		Port:           8080,
		AuthEnabled:    true,
		RequestTimeout: time.Second,
		CacheTTL:       time.Minute,
		FixturesMode:   "off",
		URL44:          default44URL,
		URL223:         default223URL,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestNumericRequestDurations(t *testing.T) {
	t.Setenv("EIS_REQUEST_TIMEOUT", "3")
	t.Setenv("EIS_RETRY_DELAY", "250")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RequestTimeout != 3*time.Second {
		t.Fatalf("request timeout = %s", cfg.RequestTimeout)
	}
	if cfg.RetryDelay != 250*time.Millisecond {
		t.Fatalf("retry delay = %s", cfg.RetryDelay)
	}
}

func TestRequestDurationSuffixIsRejected(t *testing.T) {
	t.Setenv("EIS_REQUEST_TIMEOUT", "20s")
	if _, err := Load(); err == nil {
		t.Fatal("expected duration suffix validation error")
	}
}
