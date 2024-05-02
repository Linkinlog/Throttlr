package handlers

import (
	"log/slog"
	"net/http"
)

func HandleServer(l *slog.Logger) *http.ServeMux {
	m := http.NewServeMux()

	return m
}
