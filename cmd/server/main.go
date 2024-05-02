package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/linkinlog/throttlr/internal/handlers"
)

const port = ":8008"

func main() {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	l.Info("Server listening", "port", port)

	l.Error("main.go", "err", http.ListenAndServe(port, handlers.HandleServer(l)))
}
