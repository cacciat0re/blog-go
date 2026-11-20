package route

import (
	"github.com/go-chi/chi/v5"

	"gitlab.arorms.cn/arorms/applications/blog/internal/handler/api"
)

// API is the JSON API's handlers, one per resource.
type API struct {
	Article *api.ArticleAPIHandler
}

// mount registers the API resources under the /api group. Paths are relative to
// that group, and the API addresses articles by id — the slug belongs to the
// public site.
func (a API) mount(r chi.Router) {
	r.Get("/articles", a.Article.GetAll)
	r.Post("/articles", a.Article.Create)
	r.Get("/articles/{id}", a.Article.GetByID)
}
