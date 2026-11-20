package route

import (
	"github.com/go-chi/chi/v5"

	"gitlab.arorms.cn/arorms/applications/blog/internal/handler"
)

// MVC is the server-rendered site's handlers, one per page.
type MVC struct {
	Home     *handler.HomeHandler
	Articles *handler.ArticlesHandler
	Gallery  *handler.GalleryHandler
	About    *handler.AboutHandler
}

// mount registers the pages at the site root.
func (m MVC) mount(r chi.Router) {
	r.Get("/", m.Home.Home)
	r.Get("/articles", m.Articles.List)
	r.Get("/articles/{slug}", m.Articles.Detail)
	r.Get("/gallery", m.Gallery.Gallery)
	r.Get("/about", m.About.About)
}
