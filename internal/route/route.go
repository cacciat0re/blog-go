package route

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"gitlab.arorms.cn/arorms/applications/blog/internal/config"
	"gitlab.arorms.cn/arorms/applications/blog/internal/handler"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
)

// Handlers is everything the router mounts, grouped by frontend.
type Handlers struct {
	MVC MVC
	API API
}

// New assembles the two route groups.
func New(handlers Handlers, cfg config.CORSConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	// Embedded stylesheet, scripts and fonts.
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServerFS(view.Assets())))

	// Crawlers only read robots.txt from the site root.
	r.Get("/robots.txt", robots)

	// The JSON API keeps its own plain-text 404.
	r.Route("/api", func(r chi.Router) {
		handlers.API.mount(r)
		r.NotFound(http.NotFound)
	})

	// The server-rendered site lives at the root.
	handlers.MVC.mount(r)

	// Only reachable for a request that matched no page route.
	r.NotFound(handler.NotFound)

	return r
}
