package handlers

import (
	"context"
	"log/slog"
	"net/http"

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

	m.Handle("GET /about", handleView(shared.NewLayout(pages.NewAbout(), ""), l, gs))
	m.Handle("GET /sign-up", handleView(shared.NewLayout(pages.NewAuth(false), ""), l, gs))
	m.Handle("GET /sign-in", handleView(shared.NewLayout(pages.NewAuth(true), ""), l, gs))
	m.Handle("GET /docs", handleView(shared.NewLayout(pages.NewWip(), ""), l, gs))
	m.Handle("GET /settings", handleView(shared.NewLayout(pages.NewWip(), ""), l, gs))

	m.Handle("GET /auth/", http.StripPrefix("/auth", HandleAuth(l, us, gs)))

	// catch-all + landing
	m.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		handler := handleView(shared.NewLayout(pages.NewNotFound(), ""), l, gs)
		if r.URL.Path == "/" {
			handler = handleView(shared.NewLayout(pages.NewLanding(), ""), l, gs)
		}
		handler(w, r)
	})

	return m
}

func handleView(view shared.Viewer, l *slog.Logger, gs sessions.Store) http.HandlerFunc {
	content := view.View()

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		usr, err := models.UserFromSession(r, gs)
		if err == nil {
			ctx = context.WithValue(r.Context(), models.UserCtxKey, usr)
		}

		rErr := content.Render(ctx, w)
		if rErr != nil {
			l.Error("failed to render view", "error", rErr.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
