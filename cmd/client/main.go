package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/handlers"
)

const (
	driver = "sqlite"
)

var (
	port        = "8080"
	dsn         = "throttlr.db"
	env         = internal.DefaultEnv
	callbackUrl = "http://localhost" + port
)

func init() {
	if p, err := env.Get("CLIENT_PORT"); err == nil {
		port = p
	}
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

	sqlDb, err := sql.Open(driver, dsn)
	if err != nil {
		s.Error("failed to open database", "err", err)
		return
	}
	defer sqlDb.Close()

	gs, err := setupSessions()
	if err != nil {
		s.Error("failed to setup sessions", "err", err)
		return
	}

	us := db.NewUserStore(sqlDb)
	s.Error("main.go", "err", http.ListenAndServe(":" + port, handlers.HandleClient(s, us, gs)))
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
