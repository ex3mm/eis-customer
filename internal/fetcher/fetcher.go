package fetcher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"eis-customer/internal/config"
)

const maxResponseBytes = 20 << 20

type ErrorKind string

const (
	ErrorNetwork   ErrorKind = "network"
	ErrorTimeout   ErrorKind = "timeout"
	ErrorBlocked   ErrorKind = "blocked"
	ErrorCaptcha   ErrorKind = "captcha"
	ErrorStatus    ErrorKind = "status"
	ErrorEmptyBody ErrorKind = "empty_body"
	ErrorTooLarge  ErrorKind = "too_large"
	ErrorFixture   ErrorKind = "fixture"
)

type Error struct {
	Kind       ErrorKind
	StatusCode int
	Transient  bool
	Proxy      string
	Cause      error
}

func (e *Error) Error() string {
	message := string(e.Kind)
	if e.StatusCode != 0 {
		message = fmt.Sprintf("%s (HTTP %d)", message, e.StatusCode)
	}
	if e.Proxy != "" {
		message += " via proxy " + e.Proxy
	}
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *Error) Unwrap() error { return e.Cause }

type Fetcher struct {
	timeout      time.Duration
	retryDelay   time.Duration
	maxRetries   int
	userAgent    string
	proxies      []string
	tlsConfig    *tls.Config
	logger       *slog.Logger
	proxyIndex   atomic.Uint64
	directClient *http.Client
	fixturesMode string
	fixturesDir  string
}

func New(cfg config.Config, logger *slog.Logger) (*Fetcher, error) {
	tlsConfig, err := makeTLSConfig(cfg.InsecureSkipVerify, cfg.CACertsDir)
	if err != nil {
		return nil, err
	}
	if cfg.InsecureSkipVerify {
		logger.Warn("TLS certificate verification is disabled for EIS requests")
	}
	fetcher := &Fetcher{
		timeout:      cfg.RequestTimeout,
		retryDelay:   cfg.RetryDelay,
		maxRetries:   cfg.MaxRetries,
		userAgent:    cfg.UserAgent,
		proxies:      append([]string(nil), cfg.Proxies...),
		tlsConfig:    tlsConfig,
		logger:       logger,
		fixturesMode: cfg.FixturesMode,
		fixturesDir:  cfg.FixturesDir,
	}
	fetcher.directClient = fetcher.newClient("")
	return fetcher, nil
}

func (f *Fetcher) Fetch(ctx context.Context, source, inn, kpp, rawURL string) ([]byte, error) {
	if f.fixturesMode == "read" {
		return f.readFixture(source, inn, kpp)
	}

	var lastErr error
	for attempt := 0; attempt <= f.maxRetries; attempt++ {
		if attempt > 0 {
			f.logger.Warn("retrying temporary EIS error", "attempt", attempt+1, "url", rawURL)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(f.retryDelay):
			}
		}

		body, err := f.fetchOnce(ctx, rawURL)
		if err == nil {
			if f.fixturesMode == "write" {
				if writeErr := f.writeFixture(source, inn, kpp, body); writeErr != nil {
					return nil, writeErr
				}
			}
			return body, nil
		}
		lastErr = err
		var fetchErr *Error
		if !errors.As(err, &fetchErr) || !fetchErr.Transient {
			break
		}
	}
	return nil, lastErr
}

func (f *Fetcher) readFixture(source, inn, kpp string) ([]byte, error) {
	path, err := f.fixturePath(source, inn, kpp)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, &Error{Kind: ErrorFixture, Cause: fmt.Errorf("read %s: %w", path, err)}
	}
	if len(body) == 0 {
		return nil, &Error{Kind: ErrorFixture, Cause: fmt.Errorf("fixture %s is empty", path)}
	}
	return body, nil
}

func (f *Fetcher) writeFixture(source, inn, kpp string, body []byte) error {
	path, err := f.fixturePath(source, inn, kpp)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return &Error{Kind: ErrorFixture, Cause: fmt.Errorf("create fixture directory: %w", err)}
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".fixture-*.tmp")
	if err != nil {
		return &Error{Kind: ErrorFixture, Cause: fmt.Errorf("create temporary fixture: %w", err)}
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(body); err != nil {
		_ = temporary.Close()
		return &Error{Kind: ErrorFixture, Cause: fmt.Errorf("write temporary fixture: %w", err)}
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return &Error{Kind: ErrorFixture, Cause: fmt.Errorf("set fixture permissions: %w", err)}
	}
	if err := temporary.Close(); err != nil {
		return &Error{Kind: ErrorFixture, Cause: fmt.Errorf("close temporary fixture: %w", err)}
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return &Error{Kind: ErrorFixture, Cause: fmt.Errorf("publish %s: %w", path, err)}
	}
	f.logger.Info("EIS fixture saved", "source", source, "path", path)
	return nil
}

func (f *Fetcher) fixturePath(source, inn, kpp string) (string, error) {
	if source != "fz44" && source != "fz223" {
		return "", &Error{Kind: ErrorFixture, Cause: fmt.Errorf("unsupported source %q", source)}
	}
	return filepath.Join(f.fixturesDir, source, inn+"_"+kpp+".html"), nil
}

func (f *Fetcher) fetchOnce(ctx context.Context, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, &Error{Kind: ErrorNetwork, Cause: err}
	}
	request.Header.Set("User-Agent", f.userAgent)
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")

	proxy := f.nextProxy()
	client := f.directClient
	if proxy != "" {
		client = f.newClient(proxy)
		f.logger.Debug("EIS request via proxy", "proxy", sanitizeProxy(proxy))
	}

	response, err := client.Do(request)
	if err != nil {
		kind := ErrorNetwork
		if isTimeout(err) {
			kind = ErrorTimeout
		}
		return nil, &Error{Kind: kind, Transient: true, Proxy: sanitizeProxy(proxy), Cause: err}
	}
	defer response.Body.Close()

	limited := io.LimitReader(response.Body, maxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, &Error{Kind: ErrorNetwork, Transient: true, Proxy: sanitizeProxy(proxy), Cause: err}
	}
	if len(body) > maxResponseBytes {
		return nil, &Error{Kind: ErrorTooLarge, StatusCode: response.StatusCode}
	}
	if response.StatusCode == http.StatusForbidden {
		kind := ErrorBlocked
		if looksLikeCaptcha(body) {
			kind = ErrorCaptcha
		}
		return nil, &Error{Kind: kind, StatusCode: response.StatusCode, Proxy: sanitizeProxy(proxy)}
	}
	if response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return nil, &Error{Kind: ErrorStatus, StatusCode: response.StatusCode, Transient: true, Proxy: sanitizeProxy(proxy)}
	}
	if response.StatusCode != http.StatusOK {
		return nil, &Error{Kind: ErrorStatus, StatusCode: response.StatusCode, Proxy: sanitizeProxy(proxy)}
	}
	if len(body) == 0 {
		return nil, &Error{Kind: ErrorEmptyBody, Transient: true, Proxy: sanitizeProxy(proxy)}
	}
	if looksLikeCaptcha(body) {
		return nil, &Error{Kind: ErrorCaptcha, StatusCode: response.StatusCode, Proxy: sanitizeProxy(proxy)}
	}
	return body, nil
}

func (f *Fetcher) nextProxy() string {
	if len(f.proxies) == 0 {
		return ""
	}
	index := f.proxyIndex.Add(1) - 1
	return f.proxies[int(index)%len(f.proxies)]
}

func (f *Fetcher) newClient(proxy string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = f.tlsConfig.Clone()
	if proxy != "" {
		proxyURL, _ := url.Parse(proxy)
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{
		Timeout:   f.timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

func makeTLSConfig(insecure bool, certsDir string) (*tls.Config, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if certsDir != "" {
		loaded := 0
		err = filepath.WalkDir(certsDir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			extension := strings.ToLower(filepath.Ext(path))
			if extension != ".crt" && extension != ".pem" {
				return nil
			}
			pem, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if pool.AppendCertsFromPEM(pem) {
				loaded++
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("load TLS CA certificates: %w", err)
		}
		if loaded == 0 {
			return nil, fmt.Errorf("no .crt/.pem certificates loaded from %s", certsDir)
		}
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, InsecureSkipVerify: insecure}, nil
}

func sanitizeProxy(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid-proxy]"
	}
	u.User = nil
	return u.String()
}

func looksLikeCaptcha(body []byte) bool {
	text := strings.ToLower(string(body))
	for _, marker := range []string{"captcha", "капча", "checkcaptcha", "подтвердите, что вы не робот"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
