package router

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"gitlab.arorms.cn/arorms/applications/blog/internal/config"
	"gitlab.arorms.cn/arorms/applications/blog/internal/handler"
	"gitlab.arorms.cn/arorms/applications/blog/internal/i18n"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
)

// Handlers is everything the router mounts.
type Handlers struct {
	Article  *handler.ArticleHandler
	Page     *handler.PageHandler
	Renderer *view.Renderer
}

func New(handlers Handlers, cfg config.CORSConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(canonicalLang)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	r.Get("/", redirectToDefaultLanguage)

	// Embedded stylesheet and fonts.
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServerFS(view.Assets())))

	// The two frontends, told apart by their first path segment.
	//
	// /api belongs to the separate admin application and answers JSON, so it keeps
	// its own plain-text 404 exactly as it did before the page frontend existed.
	// Everything else is the server-rendered site under /{lang}. The split is
	// structural: a path is either API or page, never both, and no route pattern
	// has to encode the difference.
	r.Route("/api", func(r chi.Router) {
		// Deliberately untouched: the contract of the admin application.
		r.Get("/articles/{slug}", handlers.Article.GetBySlug)
		r.Post("/articles", handlers.Article.GetAll)
		r.Post("/admin/articles", handlers.Article.Create)
		r.Get("/admin/articles/", handlers.Article.GetByID)
		r.NotFound(http.NotFound)
	})

	r.Route("/{lang}", func(r chi.Router) {
		r.Use(requireLanguage(handlers.Renderer))

		r.Get("/", handlers.Page.Home)
		r.Get("/articles", handlers.Page.List)
		r.Get("/articles/{slug}", handlers.Page.Detail)
		r.Get("/archive", handlers.Page.Archive)
		r.Get("/gallery", handlers.Page.Gallery)
		r.Get("/about", handlers.Page.About)
		r.NotFound(handlers.Renderer.NotFoundHandler())
	})

	// Only reachable for a request that matched neither group.
	r.NotFound(handlers.Renderer.NotFoundHandler())

	return r
}

// requireLanguage rejects a language segment that is not a supported code, so
// /fr/… and /anything really are 404s instead of English pages served under a
// made-up address.
//
// The check is middleware rather than a route constraint: /api is already kept
// out of this group by chi's static-before-parameter precedence, so all this has
// to do is validate. Putting it here also keeps the route table readable — the
// page group is a plain /{lang} prefix.
func requireLanguage(renderer *view.Renderer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := i18n.Normalize(chi.URLParam(r, "lang")); !ok {
				renderer.RenderError(w, i18n.Default(), http.StatusNotFound, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// redirectToDefaultLanguage sends a bare / to the default language.
func redirectToDefaultLanguage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, i18n.Default().BasePath(), http.StatusPermanentRedirect)
}

// canonicalLang lowercases the language segment so one page has one address.
// /EN/… and /en/… would otherwise be two URLs with identical content.
func canonicalLang(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		segs := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if len(segs) > 0 {
			if l, ok := i18n.Normalize(segs[0]); ok && segs[0] != l.String() {
				segs[0] = l.String()
				target := "/" + strings.Join(segs, "/")
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusPermanentRedirect)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
