package handlers

import (
	"log/slog"
	"net/http"

	"github.com/linkinlog/throttlr/assets"
	"github.com/linkinlog/throttlr/web/pages"
	"github.com/linkinlog/throttlr/web/shared"
)

func New(l *slog.Logger, sm SecretManager) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.NewAssets()))))

	m.Handle("GET /about", handleView(shared.NewLayout(pages.NewAbout(), ""), l))
	m.Handle("GET /sign-up", handleView(shared.NewLayout(pages.NewAuth(false), ""), l))
	m.Handle("GET /sign-in", handleView(shared.NewLayout(pages.NewAuth(true), ""), l))
	m.Handle("GET /docs", handleView(shared.NewLayout(pages.NewWip(), ""), l))

	m.Handle("GET /auth/", http.StripPrefix("/auth", HandleAuth(l, sm)))

	// catch-all + landing
	m.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		handler := handleView(shared.NewLayout(pages.NewNotFound(), ""), l)
		if r.URL.Path == "/" {
			handler = handleView(shared.NewLayout(pages.NewLanding(), ""), l)
		}
		handler(w, r)
	})

	return m
}

func handleView(view shared.Viewer, l *slog.Logger) http.HandlerFunc {
	content := view.View()

	return func(w http.ResponseWriter, r *http.Request) {
		err := content.Render(r.Context(), w)
		if err != nil {
			l.Error("failed to render view", "error", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
