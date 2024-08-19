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
	port        = "8080"
	dsn         = "postgres://username:password@localhost:5432/database_name"
	env         = internal.DefaultEnv
	callbackUrl = "http://localhost" + port
)

func init() {
	// todo move these to env
	if d, err := env.Get("CLIENT_DB"); err == nil {
		dsn = d
	}

	if url, err := env.Get("CLIENT_CALLBACK_URL"); err == nil {
		callbackUrl = url
	}
}

func main() {
	s := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s.Info("Server listening", "port", port)

	if err := setupAuth(); err != nil {
		s.Error("failed to setup auth", "err", err)
		return
	}

	sqlDb, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		s.Error("failed to open database", "err", err)
		return
	}
	defer sqlDb.Close()

	s.Error("main.go", "err", http.ListenAndServe(":"+port, handlers.HandleClient(s, sqlDb)))
}

func setupAuth() error {
	if err := internal.SetupGothic(callbackUrl); err != nil {
		return err
	}

	return nil
}
