package model

import "time"

type CheckRequest struct {
	INN string `json:"inn"`
	KPP string `json:"kpp"`
}

type SourceResult struct {
	Found bool `json:"found"`
}

type CheckResponse struct {
	INN       string                  `json:"inn"`
	KPP       string                  `json:"kpp"`
	Found     bool                    `json:"found"`
	Sources   map[string]SourceResult `json:"sources"`
	CheckedAt time.Time               `json:"checked_at"`
}
