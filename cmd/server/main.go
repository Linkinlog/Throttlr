package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/handlers"
)

var (
	port = "8081"
	dsn  = "postgres://username:password@localhost:5432/database_name"
	env  = internal.DefaultEnv
)

func init() {
	if d, err := env.Get("SERVER_DB"); err == nil {
		dsn = d
	}
}

func main() {
	s := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s.Info("Server listening", "port", port)

	sqlDb, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		s.Error("failed to open database", "err", err)
		return
	}
	defer sqlDb.Close()

	mux := handlers.HandleServer(s, sqlDb)
	s.Error("main.go", "err", http.ListenAndServe(":"+port, mux))
}
