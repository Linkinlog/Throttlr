package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/linkinlog/throttlr/internal/handlers"
)

func main() {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	fmt.Println("Server is running")
	l.Error("main.go", "err", http.ListenAndServe(":8080", handlers.New(l)))
}
