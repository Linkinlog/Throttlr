package handlers

import "net/http"

func New() http.Handler {
	m := http.NewServeMux()

	m.Handle("GET /", NewViewHandler().HandleLanding())

	return m
}
