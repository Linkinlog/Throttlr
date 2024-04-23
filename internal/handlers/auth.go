package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/markbates/goth/gothic"
)

type SecretManager interface {
	Get(key string) (string, error)
}

func HandleAuth(l *slog.Logger, sm SecretManager) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("GET /sign-out", handleLogout())
	m.Handle("GET /{provider}", handleProvider())
	m.Handle("GET /{provider}/callback", handleProviderCallback())

	return m
}

func handleProvider() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// backwards compatibility
		q := r.URL.Query()
		q.Add("provider", r.PathValue("provider"))
		r.URL.RawQuery = q.Encode()

		if _, err := gothic.CompleteUserAuth(w, r); err == nil {
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		} else {
			gothic.BeginAuthHandler(w, r)
		}
	}
}

func handleProviderCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := gothic.Logout(w, r)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}
