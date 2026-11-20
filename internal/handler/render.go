package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// The rendering half every page shares, set once by Init. It is package state
// for the same reason wakapi keeps its parsed templates here: a page handler
// holds the services it reads from, not the machinery for printing.
var (
	renderer *view.Renderer
	zone     *time.Location
)

// Init wires the template renderer and the display timezone. It must run before
// the router serves a request.
func Init(r *view.Renderer, z *time.Location) {
	renderer, zone = r, z
}

// serve is the single adapter between a controller and the renderer. It is the
// only place that maps an outcome to a response, so no controller repeats it and
// none of them writes a body.
func serve(w http.ResponseWriter, r *http.Request, controller func() (view.Page, error)) {
	page, err := controller()
	if err != nil {
		status := http.StatusInternalServerError
		var httpErr *view.HTTPError
		if errors.As(err, &httpErr) {
			status = httpErr.Status
		}
		if status >= http.StatusInternalServerError {
			log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		}
		renderer.RenderError(w, status, err)
		return
	}

	if err := renderer.Render(w, page); err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		renderer.RenderError(w, http.StatusInternalServerError, err)
	}
}

// renderError answers with the shared error page. It is the entry point for the
// route-level guards, which have no controller to go through.
func renderError(w http.ResponseWriter, status int) {
	renderer.RenderError(w, status, nil)
}

// NotFound serves the 404 page for any path outside the routes the site owns.
func NotFound(w http.ResponseWriter, r *http.Request) {
	renderError(w, http.StatusNotFound)
}

// meta builds head data for a page: the title is wrapped in the site-title
// pattern and the description falls back to the site description.
func meta(path, title, description string) viewmodel.Meta {
	return viewmodel.Meta{
		SiteName:    view.SiteName,
		Title:       fmt.Sprintf(view.TitleFormat, title),
		Description: viewmodel.Describe(description, view.SiteDescription),
		Path:        path,
	}
}

// rootMeta is the landing page's head: its title is the site itself, so it is
// not wrapped in the site-title pattern.
func rootMeta(path string) viewmodel.Meta {
	m := meta(path, view.SiteName, view.SiteDescription)
	m.Title = view.SiteName
	return m
}

// summaries converts list rows into view rows.
func summaries(items []payload.ArticleListItemResponse, base string) []viewmodel.ArticleSummaryViewModel {
	out := make([]viewmodel.ArticleSummaryViewModel, 0, len(items))
	for i, item := range items {
		out = append(out, summary(item, i, base))
	}
	return out
}

func summary(item payload.ArticleListItemResponse, index int, base string) viewmodel.ArticleSummaryViewModel {
	iso, display := viewmodel.FormatDate(item.CreatedAt, zone)
	v := viewmodel.ArticleSummaryViewModel{
		ID:          item.ID,
		Slug:        item.Slug,
		Title:       item.Title,
		Summary:     item.Summary,
		Href:        base + "/" + item.Slug,
		Index:       viewmodel.IndexOf(index),
		DateISO:     iso,
		DateDisplay: display,
	}
	if item.Category != nil {
		v.Category = &viewmodel.Link{
			Label: item.Category.Name,
			Href:  base + "?category=" + item.Category.Slug,
		}
	}
	for _, tag := range item.Tags {
		v.Tags = append(v.Tags, viewmodel.Link{Label: tag.Name, Href: base + "?tag=" + tag.Slug})
	}
	return v
}
