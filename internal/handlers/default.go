package handlers

import (
	"net/http"

	"github.com/linkinlog/throttlr/assets"
)

func New() http.Handler {
	m := http.NewServeMux()

	m.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.NewAssets()))))

	m.Handle("GET /", NewViewHandler().HandleLanding())

	return m
}
