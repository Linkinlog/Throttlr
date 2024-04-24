package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/linkinlog/throttlr/web/pages"
	"github.com/linkinlog/throttlr/web/shared"
	"github.com/markbates/goth/gothic"
)

type SecretManager interface {
	Get(key string) (string, error)
}

func HandleAuth(l *slog.Logger, sm SecretManager) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("GET /", handleView(shared.NewLayout(pages.NewNotFound(), ""), l))
	m.Handle("GET /sign-out", logHandler(l, handleLogout()))
	m.Handle("GET /{provider}", logHandler(l, handleProvider()))
	m.Handle("GET /{provider}/callback", logHandler(l, handleProviderCallback()))

	return m
}

type LogHandlerFunc func(http.ResponseWriter, *http.Request) error

func logHandler(l *slog.Logger, h LogHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			l.Error("handler error", "error", err.Error())
			handler := handleView(shared.NewLayout(pages.NewNotFound(), err.Error()), l)
			handler(w, r)
		}
	}
}

func handleProvider() LogHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.URL.Query()
		q.Add("provider", r.PathValue("provider"))
		r.URL.RawQuery = q.Encode()

		if _, err := gothic.CompleteUserAuth(w, r); err == nil {
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return nil
		} else {
			// would be nice gothic had set errors but oh well
			if !strings.Contains(err.Error(), "could not find a matching session") && !strings.EqualFold(err.Error(), "state token mismatch") {
				return errors.New("authentication failed, please try again")
			}

			gothic.BeginAuthHandler(w, r)
		}
		return nil
	}
}

func handleProviderCallback() LogHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		_, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			return err
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return nil
	}
}

func handleLogout() LogHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := gothic.Logout(w, r)
		if err != nil {
			return err
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return nil
	}
}
