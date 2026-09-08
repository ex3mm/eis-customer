package api

import (
	"encoding/json"
	"net/http"
)

type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

func writeProblem(w http.ResponseWriter, problem Problem) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(problem)
}

func badRequest(path, detail string) Problem {
	return Problem{Type: "urn:eis-customer:bad-request", Title: "Некорректный запрос", Status: http.StatusBadRequest, Detail: detail, Instance: path}
}

func unauthorized(path string) Problem {
	return Problem{Type: "urn:eis-customer:unauthorized", Title: "Требуется авторизация", Status: http.StatusUnauthorized, Detail: "Передайте корректный Bearer-токен", Instance: path}
}

func upstreamUnavailable(path string) Problem {
	return Problem{Type: "urn:eis-customer:upstream-unavailable", Title: "Источник ЕИС временно недоступен", Status: http.StatusBadGateway, Detail: "Не удалось достоверно проверить оба реестра ЕИС", Instance: path}
}
