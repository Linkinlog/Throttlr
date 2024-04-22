package handlers

import (
	"log/slog"
	"net/http"

	"github.com/linkinlog/throttlr/assets"
	"github.com/linkinlog/throttlr/web/pages"
	"github.com/linkinlog/throttlr/web/shared"
)

func New(l *slog.Logger) http.Handler {
	m := http.NewServeMux()

	m.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.NewAssets()))))

	m.Handle("GET /", handleView(shared.NewLayout(pages.Landing{}), l))
	m.Handle("GET /about", handleView(shared.NewLayout(pages.About{}), l))

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
