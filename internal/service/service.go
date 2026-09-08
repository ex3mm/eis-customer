package service

import (
	"context"
	"fmt"
	"time"

	"eis-customer/internal/model"
	"eis-customer/internal/parser"
	"eis-customer/internal/urlbuilder"
)

type Fetcher interface {
	Fetch(ctx context.Context, source, inn, kpp, rawURL string) ([]byte, error)
}

type Service struct {
	fetcher  Fetcher
	build44  urlbuilder.Builder
	build223 urlbuilder.Builder
	now      func() time.Time
}

func New(fetcher Fetcher, build44, build223 urlbuilder.Builder) *Service {
	return &Service{fetcher: fetcher, build44: build44, build223: build223, now: time.Now}
}

type sourceJob struct {
	name string
	url  string
}

type sourceAnswer struct {
	name  string
	found bool
	err   error
}

func (s *Service) Check(ctx context.Context, inn, kpp string) (model.CheckResponse, error) {
	url44, err := s.build44.Build(inn, kpp)
	if err != nil {
		return model.CheckResponse{}, err
	}
	url223, err := s.build223.Build(inn, kpp)
	if err != nil {
		return model.CheckResponse{}, err
	}

	answers := make(chan sourceAnswer, 2)
	for _, job := range []sourceJob{{name: "fz44", url: url44}, {name: "fz223", url: url223}} {
		job := job
		go func() {
			body, fetchErr := s.fetcher.Fetch(ctx, job.name, inn, kpp, job.url)
			if fetchErr != nil {
				answers <- sourceAnswer{name: job.name, err: fetchErr}
				return
			}
			found, parseErr := parser.Found(body, inn, kpp)
			answers <- sourceAnswer{name: job.name, found: found, err: parseErr}
		}()
	}

	response := model.CheckResponse{
		INN:       inn,
		KPP:       kpp,
		Sources:   make(map[string]model.SourceResult, 2),
		CheckedAt: s.now().UTC(),
	}
	for range 2 {
		answer := <-answers
		if answer.err != nil {
			return model.CheckResponse{}, fmt.Errorf("%s check failed: %w", answer.name, answer.err)
		}
		response.Sources[answer.name] = model.SourceResult{Found: answer.found}
		response.Found = response.Found || answer.found
	}
	return response, nil
}
