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
	level := slog.LevelInfo
	if internal.Debug() {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{AddSource: true, Level: level}
	l := slog.New(slog.NewTextHandler(os.Stdout, opts))
	l.Info("Server listening", "port", port)

	sqlDb, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		l.Error("failed to open database", "err", err)
		return
	}
	defer sqlDb.Close()

	mux := handlers.HandleServer(l, sqlDb)
	l.Error("main.go", "err", http.ListenAndServe(":"+port, mux))
}
