package handler

import (
	"net/http"

	"gitlab.arorms.cn/arorms/applications/blog/internal/service"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// homeLatest is how many articles the landing page shows, matching the legacy
// homepage which sliced the first four rows of a six-row query.
const homeLatest = 4

// homeQuerySize is how many rows the landing page asks for; the legacy client
// requested six so it could show four plus its own headings.
const homeQuerySize = 6

// HomeHandler serves the landing page: the masthead and the latest articles.
type HomeHandler struct {
	articles service.IArticleService
}

func NewHomeHandler(articles service.IArticleService) *HomeHandler {
	return &HomeHandler{articles: articles}
}

// Home renders the landing page.
func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	serve(w, r, func() (view.Page, error) {
		query := viewmodel.ListQuery{Page: 1, Limit: homeQuerySize}

		result, err := h.articles.GetAllPublished(r.Context(), query.Payload())
		if err != nil {
			return view.Page{}, err
		}
		rows := result.Content
		if len(rows) > homeLatest {
			rows = rows[:homeLatest]
		}

		// Rows link into the articles section.
		base := viewmodel.SectionPath("articles")
		return view.Page{
			View: view.HomeView,
			Data: viewmodel.HomeViewModel{
				Meta:     rootMeta(r.URL.Path),
				Articles: summaries(rows, base),
			},
		}, nil
	})
}
