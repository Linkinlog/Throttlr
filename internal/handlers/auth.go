package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	"github.com/linkinlog/throttlr/web/pages"
	"github.com/linkinlog/throttlr/web/shared"
	"github.com/markbates/goth/gothic"
)

func HandleAuth(l *slog.Logger, us *db.UserStore) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("GET /", handleView(shared.NewLayout(pages.NewNotFound(), ""), l))
	m.Handle("GET /sign-out", logHandler(l, handleLogout()))
	m.Handle("GET /{provider}", logHandler(l, handleProvider(us)))
	m.Handle("GET /{provider}/callback", logHandler(l, handleProviderCallback(us)))

	return m
}

type httpError struct {
	error
	display string
}

type HandlerErrorFunc func(http.ResponseWriter, *http.Request) *httpError

func logHandler(l *slog.Logger, h HandlerErrorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpErr := h(w, r)
		if httpErr != nil {
			l.Error("handler error", "error", httpErr.Error())
			handler := handleView(shared.NewLayout(pages.NewNotFound(), httpErr.display), l)
			handler(w, r)
		}
	}
}

func handleProvider(us *db.UserStore) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		q := r.URL.Query()
		q.Add("provider", r.PathValue("provider"))
		r.URL.RawQuery = q.Encode()

		if u, err := gothic.CompleteUserAuth(w, r); err == nil {
			uId := fmt.Sprintf("%s-%s", u.UserID, u.Provider)
			if _, err := us.ById(r.Context(), uId); err == nil {
				// @TODO
			} else {
				usr := models.NewUser().
					SetId(uId).
					SetName(u.Name).
					SetEmail(u.Email)

				if err := us.Store(r.Context(), *usr); err != nil {
					return &httpError{
						error:   err,
						display: "user creation failed, try again.",
					}
				}

				// @TODO
			}
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return nil
		} else {
			// would be nice gothic had set errors but oh well
			if !strings.Contains(err.Error(), "could not find a matching session") &&
				!strings.EqualFold(err.Error(), "state token mismatch") {
				return &httpError{
					error:   err,
					display: "auth failed, try again.",
				}
			}

			gothic.BeginAuthHandler(w, r)
		}
		return nil
	}
}

func handleProviderCallback(us *db.UserStore) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		u, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			return &httpError{
				error:   err,
				display: "auth failed, try again.",
			}
		}
		uId := fmt.Sprintf("%s-%s", u.UserID, u.Provider)
		if _, err := us.ById(r.Context(), uId); err == nil {
			// @TODO
		} else {
			usr := models.NewUser().
				SetId(uId).
				SetName(u.Name).
				SetEmail(u.Email)

			if err := us.Store(r.Context(), *usr); err != nil {
				return &httpError{
					error:   err,
					display: "user creation failed, try again.",
				}
			}

			// @TODO
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return nil
	}
}

func handleLogout() HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		err := gothic.Logout(w, r)
		if err != nil {
			return &httpError{
				error:   err,
				display: "logout failed, try again.",
			}
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return nil
	}
}
