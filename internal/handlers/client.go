package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/sessions"
	"github.com/linkinlog/throttlr/assets"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	"github.com/linkinlog/throttlr/web/pages"
	"github.com/linkinlog/throttlr/web/shared"
)

func HandleClient(l *slog.Logger, us *db.UserStore, gs sessions.Store) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.NewAssets()))))

	m.Handle("GET /about", withUser(handleView(shared.NewLayout(pages.NewAbout(), ""), l), gs))
	m.Handle("GET /sign-up", withUser(handleView(shared.NewLayout(pages.NewAuth(false), ""), l), gs))
	m.Handle("GET /sign-in", withUser(handleView(shared.NewLayout(pages.NewAuth(true), ""), l), gs))
	m.Handle("GET /docs", withUser(handleView(shared.NewLayout(pages.NewWip(), ""), l), gs))
	m.Handle("GET /settings", withUser(handleView(shared.NewLayout(pages.NewSettings(), ""), l), gs))

	m.Handle("GET /endpoints", withUser(handleView(shared.NewLayout(pages.NewEndpointForm(), ""), l), gs))
	m.Handle("GET /views/endpoints", proxyToServer())
	m.Handle("POST /register/{apiKey}", proxyToServer())

	m.Handle("GET /auth/", http.StripPrefix("/auth", HandleAuth(l, us, gs)))

	// catch-all + landing
	m.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		handler := handleView(shared.NewLayout(pages.NewNotFound(), ""), l)
		if r.URL.Path == "/" {
			_, err := models.UserFromSession(r, gs)
			if err == nil {
				handler = withUser(handleView(shared.NewLayout(pages.NewDashboard(), ""), l), gs)
			} else {
				handler = handleView(shared.NewLayout(pages.NewLanding(), ""), l)
			}
		}
		handler(w, r)
	})

	return m
}

func proxyToServer() http.HandlerFunc {
	callbackUrl := "http://server:8081"
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(callbackUrl)
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
