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

var port = "8080"

func main() {
	s := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s.Info("Server listening", "port", port)

	if err := setupAuth(); err != nil {
		s.Error("failed to setup auth", "err", err)
		return
	}

	dsn := internal.ClientDB()

	sqlDb, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		s.Error("failed to open database", "err", err)
		return
	}
	defer sqlDb.Close()

	mux := handlers.HandleClient(s, sqlDb)
	s.Error("main.go", "err", http.ListenAndServe(":"+port, mux))
}

func setupAuth() error {
	callbackUrl := internal.ClientCallbackURL()
	if err := internal.SetupGothic(callbackUrl); err != nil {
		return err
	}

	return nil
}
