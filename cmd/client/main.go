package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/handlers"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

const port = ":8080"

func main() {
	s := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s.Info("Server listening", "port", port)

	if err := setupGothic(); err != nil {
		s.Error(err.Error())
		return
	}

	sqlDb, err := sql.Open("sqlite", "throttlr.db")
	if err != nil {
		s.Error("failed to open database", "err", err)
		return
	}

	us := db.NewUserStore(sqlDb)

	s.Error("main.go", "err", http.ListenAndServe(port, handlers.HandleClient(s, us)))
}

func setupGothic() error {
	env := internal.NewEnv()

	ghKey, err := env.Get("GITHUB_KEY")
	if err != nil {
		return fmt.Errorf("failed to get GITHUB_KEY, %w", err)
	}
	ghSecret, err := env.Get("GITHUB_SECRET")
	if err != nil {
		return fmt.Errorf("failed to get GITHUB_SECRET, %w", err)
	}
	gk, err := env.Get("GOOGLE_KEY")
	if err != nil {
		return fmt.Errorf("failed to get GOOGLE_KEY, %w", err)
	}
	gs, err := env.Get("GOOGLE_SECRET")
	if err != nil {
		return fmt.Errorf("failed to get GOOGLE_SECRET, %w", err)
	}

	url := "http://localhost" + port

	if env, _ := env.Get("ENV"); env == "prod" {
		slog.Info("Running in production mode")
		url = "https://throttlr.dahlton.org"
	}

	goth.UseProviders(
		github.New(ghKey, ghSecret, url+"/auth/github/callback"),
		google.New(gk, gs, url+"/auth/google/callback", "https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"),
	)
	return nil
}
