package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
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

	sqlDb, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		s.Error("failed to open database", "err", err)
		return
	}
	defer sqlDb.Close(context.Background())

	kStore := db.NewKeyStore(sqlDb)

	epStore := db.NewEndpointStore(sqlDb)

	bStore := db.NewBucketStore(sqlDb)

	s.Error("main.go", "err", http.ListenAndServe(":"+port, handlers.HandleServer(s, kStore, epStore, bStore)))
}
