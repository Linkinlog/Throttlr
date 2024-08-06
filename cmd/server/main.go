package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/handlers"
)

const (
	driver = "sqlite"
)

var (
	port = "8081"
	dsn  = "throttlr.db"
	env  = internal.DefaultEnv
)

func init() {
	if p, err := env.Get("SERVER_PORT"); err == nil {
		port = p
	}
	if d, err := env.Get("SERVER_DB"); err == nil {
		dsn = d
	}
}

func main() {
	s := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s.Info("Server listening", "port", port)

	sqlDb, err := sql.Open(driver, dsn)
	if err != nil {
		s.Error("failed to open database", "err", err)
		return
	}
	defer sqlDb.Close()

	kStore := db.NewKeyStore(sqlDb)

	epStore := db.NewEndpointStore(sqlDb)

	bStore := db.NewBucketStore(sqlDb)

	s.Error("main.go", "err", http.ListenAndServe(":"+port, handlers.HandleServer(s, kStore, epStore, bStore)))
}
