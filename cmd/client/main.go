package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
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
	if d, err := env.Get("CLIENT_DB"); err == nil {
		dsn = d
	}

	if url, err := env.Get("CALLBACK_URL"); err == nil {
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

	sqlDb, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		s.Error("failed to open database", "err", err)
		return
	}
	defer sqlDb.Close(context.Background())

	gs, err := setupSessions()
	if err != nil {
		s.Error("failed to setup sessions", "err", err)
		return
	}

	s.Error("main.go", "err", http.ListenAndServe(":"+port, handlers.HandleClient(s, sqlDb, gs)))
}

func setupAuth() error {
	if err := internal.SetupGothic(callbackUrl); err != nil {
		return err
	}

	return nil
}

func setupSessions() (sessions.Store, error) {
	env := internal.DefaultEnv

	secret, err := env.Get("SESSION_SECRET")
	if err != nil {
		return nil, fmt.Errorf("failed to get SESSION_SECRET, %w", err)
	}

	gs := sessions.NewCookieStore([]byte(secret))

	return gs, nil
}
