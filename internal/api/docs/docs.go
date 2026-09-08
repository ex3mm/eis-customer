package docs

import (
	"embed"
	"net/http"
)

//go:embed openapi.yaml index.html
var assets embed.FS

func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		body, _ := assets.ReadFile("openapi.yaml")
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(body)
	})
	mux.HandleFunc("GET /docs", serveUI)
	mux.HandleFunc("GET /docs/", serveUI)
}

func serveUI(w http.ResponseWriter, _ *http.Request) {
	body, _ := assets.ReadFile("index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(body)
}
