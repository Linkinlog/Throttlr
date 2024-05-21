package handlers

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	"github.com/linkinlog/throttlr/web/pages"
	"github.com/linkinlog/throttlr/web/shared"
	"github.com/markbates/goth/gothic"
)

const (
	logoutDisplay     = "logout failed, please try again."
	authFailedDisplay = "auth failed, please try again."
)

var AuthError = errors.New("failed to authenticate user req")

func init() {
	gob.Register(models.User{})
	gob.Register(models.UserCtxKey)
}

func HandleAuth(l *slog.Logger, us *db.UserStore, gs sessions.Store) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("GET /", withUser(handleView(shared.NewLayout(pages.NewNotFound(), ""), l), gs))
	m.Handle("GET /sign-out", logHandler(l, gs, handleLogout(gs)))
	m.Handle("GET /delete", withUser(logHandler(l, gs, handleDelete(us)), gs))
	m.Handle("GET /{provider}", logHandler(l, gs, handleProvider(us, gs)))
	m.Handle("GET /{provider}/callback", logHandler(l, gs, handleProviderCallback(us, gs)))

	return m
}

type httpError struct {
	error
	display string
}

type HandlerErrorFunc func(http.ResponseWriter, *http.Request) *httpError

func logHandler(l *slog.Logger, gs sessions.Store, h HandlerErrorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpErr := h(w, r)
		if httpErr != nil {
			l.Error("handler error", "error", httpErr.Error())
			handler := withUser(handleView(shared.NewLayout(pages.NewNotFound(), httpErr.display), l), gs)
			handler(w, r)
		}
	}
}

func handleDelete(us *db.UserStore) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		user := models.UserFromCtx(r.Context())
		fmt.Println(user)
		err := us.Delete(r.Context(), user.Id)
		if err != nil {
			return &httpError{
				error:   fmt.Errorf("delete: %w", err),
				display: logoutDisplay,
			}
		}
		w.Header().Set("Location", "/auth/sign-out")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return nil
	}
}

func handleProvider(us *db.UserStore, gs sessions.Store) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		q := r.URL.Query()
		q.Add("provider", r.PathValue("provider"))
		r.URL.RawQuery = q.Encode()

		if err := internal.AuthenticateUserRequest(w, r, gs, us); err != nil {
			slog.Debug(AuthError.Error(), "error", err)
			gothic.BeginAuthHandler(w, r)
		}

		return nil
	}
}

func handleProviderCallback(us *db.UserStore, gs sessions.Store) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		if err := internal.AuthenticateUserRequest(w, r, gs, us); err != nil {
			return &httpError{
				error:   fmt.Errorf("callback: %w, %w", AuthError, err),
				display: authFailedDisplay,
			}
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return nil
	}
}

func handleLogout(gs sessions.Store) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		err := gothic.Logout(w, r)
		if err != nil {
			return &httpError{
				error:   err,
				display: logoutDisplay,
			}
		}
		rErr := models.RemoveUserFromSession(r, w, gs)
		if rErr != nil {
			return &httpError{
				error:   rErr,
				display: logoutDisplay,
			}
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return nil
	}
}
