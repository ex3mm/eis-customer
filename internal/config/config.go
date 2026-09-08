package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	default44URL  = "https://zakupki.gov.ru/epz/organization/search/results.html"
	default223URL = "https://zakupki.gov.ru/epz/customer223/search/results.html"
)

type Config struct {
	Port               int
	DocsEnabled        bool
	AuthEnabled        bool
	APIKey             string
	RequestTimeout     time.Duration
	RetryDelay         time.Duration
	MaxRetries         int
	UserAgent          string
	Proxies            []string
	InsecureSkipVerify bool
	CACertsDir         string
	CacheEnabled       bool
	CacheTTL           time.Duration
	FixturesMode       string
	FixturesDir        string
	URL44              string
	URL223             string
	Params44           map[string]string
	Params223          map[string]string
}

func Load() (Config, error) {
	cfg := Config{
		Port:               envInt("EIS_SERVER_PORT", 8080),
		DocsEnabled:        envBool("EIS_DOCS_ENABLED", true),
		AuthEnabled:        envBool("EIS_AUTH_ENABLED", false),
		APIKey:             os.Getenv("EIS_AUTH_API_KEY"),
		RequestTimeout:     envIntegerDuration("EIS_REQUEST_TIMEOUT", 20*time.Second, time.Second),
		RetryDelay:         envIntegerDuration("EIS_RETRY_DELAY", 500*time.Millisecond, time.Millisecond),
		MaxRetries:         envInt("EIS_MAX_RETRIES", 1),
		UserAgent:          envString("EIS_USER_AGENT", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/124 Safari/537.36"),
		Proxies:            envList("EIS_PROXIES"),
		InsecureSkipVerify: envBool("EIS_INSECURE_SKIP_TLS_VERIFY", false),
		CACertsDir:         os.Getenv("EIS_TLS_CA_CERTS_DIR"),
		CacheEnabled:       envBool("EIS_CACHE_ENABLED", false),
		CacheTTL:           envDuration("EIS_CACHE_TTL", time.Hour),
		FixturesMode:       strings.ToLower(envString("EIS_FIXTURES_MODE", "off")),
		FixturesDir:        envString("EIS_FIXTURES_DIR", "fixtures"),
		URL44:              envString("EIS_44_BASE_URL", default44URL),
		URL223:             envString("EIS_223_BASE_URL", default223URL),
		Params44: map[string]string{
			"morphology":                            envString("EIS_44_PARAM_MORPHOLOGY", "on"),
			"fz94":                                  envString("EIS_44_PARAM_FZ94", "on"),
			"fz223":                                 envString("EIS_44_PARAM_FZ223", "on"),
			"F":                                     envString("EIS_44_PARAM_F", "on"),
			"S":                                     envString("EIS_44_PARAM_S", "on"),
			"M":                                     envString("EIS_44_PARAM_M", "on"),
			"NOT_FSM":                               envString("EIS_44_PARAM_NOT_FSM", "on"),
			"registered94":                          envString("EIS_44_PARAM_REGISTERED94", "on"),
			"notRegistered":                         envString("EIS_44_PARAM_NOT_REGISTERED", "on"),
			"organizationRoleValueIdNameHidden":     envString("EIS_44_PARAM_ORGANIZATION_ROLE", "{}"),
			"organizationTypeListValueIdNameHidden": envString("EIS_44_PARAM_ORGANIZATION_TYPE", "{}"),
			"legalEntityKindComponentIdNameHidden":  envString("EIS_44_PARAM_LEGAL_ENTITY_KIND", "{}"),
			"sortBy":                                envString("EIS_44_PARAM_SORT_BY", "NAME"),
			"pageNumber":                            envString("EIS_44_PARAM_PAGE_NUMBER", "1"),
			"sortDirection":                         envString("EIS_44_PARAM_SORT_DIRECTION", "false"),
			"recordsPerPage":                        envString("EIS_44_PARAM_RECORDS_PER_PAGE", "_10"),
			"showLotsInfoHidden":                    envString("EIS_44_PARAM_SHOW_LOTS_INFO", "false"),
		},
		Params223: map[string]string{
			"morphology":                        envString("EIS_223_PARAM_MORPHOLOGY", "on"),
			"search-filter":                     envString("EIS_223_PARAM_SEARCH_FILTER", "Дате размещения"),
			"pageNumber":                        envString("EIS_223_PARAM_PAGE_NUMBER", "1"),
			"sortDirection":                     envString("EIS_223_PARAM_SORT_DIRECTION", "false"),
			"recordsPerPage":                    envString("EIS_223_PARAM_RECORDS_PER_PAGE", "_10"),
			"showLotsInfoHidden":                envString("EIS_223_PARAM_SHOW_LOTS_INFO", "false"),
			"sortBy":                            envString("EIS_223_PARAM_SORT_BY", "NAME"),
			"customer223Status_0":               envString("EIS_223_PARAM_REGISTERED_STATUS_ENABLED", "on"),
			"customer223Status":                 envString("EIS_223_PARAM_REGISTERED_STATUS", "0"),
			"organizationRoleValueIdNameHidden": envString("EIS_223_PARAM_ORGANIZATION_ROLE", "{}"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("EIS_SERVER_PORT must be between 1 and 65535")
	}
	if c.AuthEnabled && strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("EIS_AUTH_API_KEY is required when EIS_AUTH_ENABLED=true")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("EIS_REQUEST_TIMEOUT must be a positive integer number of seconds")
	}
	if c.RetryDelay < 0 {
		return fmt.Errorf("EIS_RETRY_DELAY must be a non-negative integer number of milliseconds")
	}
	if c.CacheTTL <= 0 {
		return fmt.Errorf("EIS_CACHE_TTL must be positive")
	}
	if c.MaxRetries < 0 || c.MaxRetries > 1 {
		return fmt.Errorf("EIS_MAX_RETRIES must be 0 or 1")
	}
	if c.FixturesMode != "off" && c.FixturesMode != "read" && c.FixturesMode != "write" {
		return fmt.Errorf("EIS_FIXTURES_MODE must be off, read or write")
	}
	if c.FixturesMode != "off" && strings.TrimSpace(c.FixturesDir) == "" {
		return fmt.Errorf("EIS_FIXTURES_DIR is required when fixture mode is enabled")
	}
	for name, raw := range map[string]string{"EIS_44_BASE_URL": c.URL44, "EIS_223_BASE_URL": c.URL223} {
		u, err := url.ParseRequestURI(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("%s must be an absolute HTTP(S) URL", name)
		}
	}
	for _, raw := range c.Proxies {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5") {
			return fmt.Errorf("invalid proxy URL %q", raw)
		}
	}
	return nil
}

func envString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envIntegerDuration(key string, fallback, unit time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed < 0 {
		return -1
	}
	return time.Duration(parsed) * unit
}

func envList(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
