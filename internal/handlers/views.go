package handlers

import (
	"net/http"

	"github.com/linkinlog/throttlr/web/pages"
	"github.com/linkinlog/throttlr/web/shared"
)

func NewViewHandler() *ViewHandler {
	return &ViewHandler{}
}

type ViewHandler struct{}

func (h *ViewHandler) HandleLanding() http.HandlerFunc {
	view := shared.NewLayout().SetContent(pages.Landing{})

	return func(w http.ResponseWriter, r *http.Request) {
		view.View().Render(r.Context(), w)
	}
}
