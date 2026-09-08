package urlbuilder

import (
	"fmt"
	"net/url"
)

type Builder struct {
	baseURL string
	params  map[string]string
}

func New(baseURL string, params map[string]string) Builder {
	copyParams := make(map[string]string, len(params))
	for key, value := range params {
		copyParams[key] = value
	}
	return Builder{baseURL: baseURL, params: copyParams}
}

// Build собирает URL поиска. ИНН/КПП всегда берутся из API-запроса,
// остальные значения задаются конфигурацией и могут быть переопределены через ENV.
func (b Builder) Build(inn, kpp string) (string, error) {
	u, err := url.Parse(b.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}
	query := u.Query()
	for key, value := range b.params {
		if value != "" {
			query.Set(key, value)
		}
	}
	query.Set("searchString", inn)
	query.Set("inn", inn)
	query.Set("kpp", kpp)
	u.RawQuery = query.Encode()
	return u.String(), nil
}
