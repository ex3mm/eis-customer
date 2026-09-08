package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"eis-customer/internal/cache"
	"eis-customer/internal/model"
	"eis-customer/internal/service"
)

var (
	innPattern = regexp.MustCompile(`^(?:\d{10}|\d{12})$`)
	kppPattern = regexp.MustCompile(`^\d{9}$`)
)

type Handler struct {
	service *service.Service
	cache   *cache.Cache
	logger  *slog.Logger
}

func NewHandler(service *service.Service, cache *cache.Cache, logger *slog.Logger) *Handler {
	return &Handler{service: service, cache: cache, logger: logger}
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	request, err := decodeRequest(w, r)
	if err != nil {
		writeProblem(w, badRequest(r.URL.Path, err.Error()))
		return
	}
	if !innPattern.MatchString(request.INN) {
		writeProblem(w, badRequest(r.URL.Path, "Поле inn должно содержать 10 или 12 цифр"))
		return
	}
	if !kppPattern.MatchString(request.KPP) {
		writeProblem(w, badRequest(r.URL.Path, "Поле kpp должно содержать ровно 9 цифр"))
		return
	}

	cacheKey := request.INN + ":" + request.KPP
	if cached, ok := h.cache.Get(cacheKey); ok {
		w.Header().Set("X-Cache", "HIT")
		writeJSON(w, cached)
		return
	}

	response, err := h.service.Check(r.Context(), request.INN, request.KPP)
	if err != nil {
		h.logger.Error("accreditation check failed", "inn", request.INN, "kpp", request.KPP, "error", err)
		writeProblem(w, upstreamUnavailable(r.URL.Path))
		return
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		h.logger.Error("encode response failed", "error", err)
		writeProblem(w, Problem{Type: "urn:eis-customer:internal", Title: "Внутренняя ошибка сервиса", Status: http.StatusInternalServerError, Instance: r.URL.Path})
		return
	}
	h.cache.Set(cacheKey, encoded)
	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, encoded)
}

func decodeRequest(w http.ResponseWriter, r *http.Request) (model.CheckRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request model.CheckRequest
	if err := decoder.Decode(&request); err != nil {
		return model.CheckRequest{}, errors.New("Тело запроса должно быть корректным JSON с полями inn и kpp")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return model.CheckRequest{}, errors.New("Тело запроса должно содержать ровно один JSON-объект")
	}
	return request, nil
}

func writeJSON(w http.ResponseWriter, encoded []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
}
