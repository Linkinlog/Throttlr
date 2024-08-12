package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/assets"
	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	"github.com/linkinlog/throttlr/web/pages"
	"github.com/linkinlog/throttlr/web/shared"
)

const serverURL string = "http://server:8081"

func HandleClient(l *slog.Logger, pool *pgx.Conn) *http.ServeMux {
	env := internal.DefaultEnv

	secret, err := env.Get("SESSION_SECRET")
	if err != nil {
		l.Error("failed to get session secret", "error", err)
		return nil
	}

	gs := sessions.NewCookieStore([]byte(secret))
	m := http.NewServeMux()

	m.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.NewAssets()))))

	m.Handle("GET /about", withUser(handleView(shared.NewLayout(pages.NewAbout(), ""), l), gs))
	m.Handle("GET /sign-up", withUser(handleView(shared.NewLayout(pages.NewAuth(false), ""), l), gs))
	m.Handle("GET /sign-in", withUser(handleView(shared.NewLayout(pages.NewAuth(true), ""), l), gs))
	m.Handle("GET /settings", withUser(handleView(shared.NewLayout(pages.NewSettings(), ""), l), gs))

	m.Handle("GET /endpoints", withUser(handleView(shared.NewLayout(pages.NewEndpointForm(), ""), l), gs))
	m.Handle("GET /views/endpoints/{throttlrPath}", withUser(handleViewEndpoint(pool, gs, l), gs))
	m.Handle("POST /v1/register", proxyToServer())
	m.Handle("POST /v1/update/{throttlrPath}", proxyToServer())
	m.Handle("POST /v1/delete/{throttlrPath}", proxyToServer())

	m.Handle("GET /auth/", http.StripPrefix("/auth", HandleAuth(l, pool, gs)))

	// catch-all + landing
	m.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		handler := handleView(shared.NewLayout(pages.NewNotFound(), ""), l)
		if r.URL.Path == "/" {
			u, err := models.UserFromSession(r, gs)
			if err == nil {
				es := db.NewEndpointStore(pool)
				endpoints, err := es.AllForUser(r.Context(), u.Id)
				if err != nil {
					l.Error("failed to get user endpoints", "error", err)
					handler(w, r)
					return
				}

				handler = withUser(handleView(shared.NewLayout(pages.NewDashboard(endpoints, u.ApiKey.String()), ""), l), gs)
			} else {
				handler = handleView(shared.NewLayout(pages.NewLanding(), ""), l)
			}
		}
		handler(w, r)
	})

	return m
}

func proxyToServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(serverURL)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(u)
		proxy.ServeHTTP(w, r)
	}
}

func withUser(next http.HandlerFunc, gs sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usr, err := models.UserFromSession(r, gs)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), models.UserCtxKey, usr)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func handleView(view shared.Viewer, l *slog.Logger) http.HandlerFunc {
	content := view.View()

	return func(w http.ResponseWriter, r *http.Request) {
		rErr := content.Render(r.Context(), w)
		if rErr != nil {
			l.Error("failed to render view", "error", rErr.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func handleViewEndpoint(pool *pgx.Conn, gs sessions.Store, l *slog.Logger) http.HandlerFunc {
	es := db.NewEndpointStore(pool)

	return func(w http.ResponseWriter, r *http.Request) {
		throttlrPath := r.PathValue("throttlrPath")
		if throttlrPath == "" {
			l.Error("no throttlr path provided")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		e := &models.Endpoint{
			Bucket:       &models.Bucket{},
			ThrottlrPath: throttlrPath,
		}

		u, err := models.UserFromSession(r, gs)
		if err != nil {
			l.Error("failed to get user from session", "error", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		err = es.Fill(r.Context(), e, u.Id)
		if err != nil {
			l.Error("failed to get endpoint", "error", err, "endpoint", throttlrPath)
			if err == pgx.ErrNoRows {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		handler := handleView(shared.NewLayout(pages.NewEndpointView(e), ""), l)

		handler(w, r)
	}
}
