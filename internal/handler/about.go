package handler

import (
	"net/http"

	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// AboutHandler serves the static about page. It has no dependencies at all.
type AboutHandler struct{}

func NewAboutHandler() *AboutHandler { return &AboutHandler{} }

// About renders the about page.
func (h *AboutHandler) About(w http.ResponseWriter, r *http.Request) {
	serve(w, r, func() (view.Page, error) {
		return view.Page{
			View: view.AboutView,
			Data: viewmodel.AboutViewModel{
				Meta: meta(r.URL.Path, "关于", view.SiteDescription),
			},
		}, nil
	})
}
