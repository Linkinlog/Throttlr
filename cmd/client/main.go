package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/linkinlog/throttlr/internal/handlers"
	"github.com/linkinlog/throttlr/internal/secrets"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

const port = ":8080"

func main() {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	url := "http://localhost" + port

	var secretManager handlers.SecretManager = secrets.NewDev()
	if os.Getenv("ENV") == "prod" {
		l.Info("using GCP secret manager")
		secretManager = secrets.NewGCP("throttlr-421001")
		url = "https://throttlr.dahlton.org"
	} else {
		l.Info("using dev secret manager")
		l.Info(os.Getenv("ENV"))
	}

	ghKey, err := secretManager.Get("GITHUB_KEY")
	if err != nil {
		l.Error("failed to get GITHUB_KEY", "error", err.Error())
	}
	ghSecret, err := secretManager.Get("GITHUB_SECRET")
	if err != nil {
		l.Error("failed to get GITHUB_SECRET", "error", err.Error())
	}
	gk, err := secretManager.Get("GOOGLE_KEY")
	if err != nil {
		l.Error("failed to get GOOGLE_KEY", "error", err.Error())
	}
	gs, err := secretManager.Get("GOOGLE_SECRET")
	if err != nil {
		l.Error("failed to get GOOGLE_SECRET", "error", err.Error())
	}

	goth.UseProviders(
		github.New(ghKey, ghSecret, url+"/auth/github/callback"),
		google.New(gk, gs, url+"/auth/google/callback"),
	)

	sessionSecret, err := secretManager.Get("SESSION_SECRET")
	if err != nil {
		l.Error("failed to get SESSION_SECRET", "error", err.Error())
	}

	gothic.Store = sessions.NewCookieStore([]byte(sessionSecret))

	fmt.Println("Server is running")
	l.Error("main.go", "err", http.ListenAndServe(port, handlers.New(l, secretManager)))
}
