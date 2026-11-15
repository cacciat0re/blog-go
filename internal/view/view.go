// Package view renders the public HTML pages with the standard library's
// html/template.
//
// A handler produces a Page value; Renderer.Render turns it into a full document.
// Template sets are parsed once at startup — one set per (language, view) pair —
// so a request only executes them.
package view

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gitlab.arorms.cn/arorms/applications/blog/internal/i18n"
	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// View names identify a template set. They are logical names, not filenames: a
// handler never needs to know which .html file ends up being used.
const (
	HomeView          = "home"
	ArticleListView   = "article-list"
	ArticleDetailView = "article-detail"
	ArchiveView       = "archive"
	GalleryView       = "gallery"
	AboutView         = "about"
	ErrorView         = "error"
)

// Page is a successful response: which view to use, and the view model to print.
type Page struct {
	View   string
	Status int // 0 means 200
	Data   any // one of the structs in internal/viewmodel
}

// The embedded file set: templates plus the assets that are served, grouped by
// kind on disk and mirrored one-to-one in the URL space.
//
// The CSS is listed file by file rather than as "assets" so the Tailwind source
// (assets/css/input.css) stays out of the binary — it is a build input, not
// something a browser should ever be able to download.
//
// assets/js is empty today; go:embed rejects an empty directory, so it is
// deliberately absent here and can be added once it holds a file.
//
//go:embed templates/*.html templates/layouts/*.html templates/partials/*.html
//go:embed assets/css/index.css assets/css/katex
//go:embed assets/img
var files embed.FS

// pageFiles maps a view name to the file that defines its "content" block.
var pageFiles = map[string]string{
	HomeView:          "templates/home.html",
	ArticleListView:   "templates/article-list.html",
	ArticleDetailView: "templates/article-detail.html",
	ArchiveView:       "templates/archive.html",
	GalleryView:       "templates/gallery.html",
	AboutView:         "templates/about.html",
	ErrorView:         "templates/error.html",
}

// Renderer holds one parsed template set per (language, view).
type Renderer struct {
	dict *i18n.Dictionary
	sets map[i18n.Lang]map[string]*template.Template
}

// New parses every template set for every supported language.
//
// The sets differ only in the translation function they close over; the HTML
// files are the same ones each time. Parsing is eager because a template error
// should stop the process at boot instead of failing a reader's page later.
func New(dict *i18n.Dictionary) (*Renderer, error) {
	r := &Renderer{
		dict: dict,
		sets: make(map[i18n.Lang]map[string]*template.Template, len(i18n.Supported())),
	}
	for _, lang := range i18n.Supported() {
		sets := make(map[string]*template.Template, len(pageFiles))
		for name, page := range pageFiles {
			t, err := template.New("layout.html").Funcs(template.FuncMap{
				"t": dict.Translator(lang),
			}).ParseFS(files,
				"templates/layouts/layout.html",
				"templates/partials/*.html",
				page,
			)
			if err != nil {
				return nil, fmt.Errorf("parse view %q (%s): %w", name, lang, err)
			}
			if t.Lookup("layout.html") == nil {
				return nil, fmt.Errorf("view %q (%s): layout template missing", name, lang)
			}
			if t.Lookup("content") == nil {
				return nil, fmt.Errorf("view %q (%s): %s does not define \"content\"", name, lang, page)
			}
			sets[name] = t
		}
		r.sets[lang] = sets
	}
	return r, nil
}

// Assets returns the embedded static files, addressed as the templates refer to
// them: /assets/index.css, /assets/katex/…, /assets/img/….
func Assets() fs.FS { return assetFS{files: files} }

// Render writes a page to w.
//
// The document is rendered into a buffer first: if template execution fails,
// nothing has been written yet and the caller can still answer with a clean 500.
// It is also why a handler returning an error can never leave half an HTML
// document on the wire.
func (r *Renderer) Render(w http.ResponseWriter, p Page) error {
	if err := r.write(w, p.Data, p.View, statusOf(p.Status)); err != nil {
		return err
	}
	return nil
}

// RenderError writes the shared error page for status. It never fails silently:
// when even the error page cannot be rendered, it falls back to a plain-text
// status body.
func (r *Renderer) RenderError(w http.ResponseWriter, lang i18n.Lang, status int, detail error) {
	if status >= http.StatusInternalServerError && detail != nil {
		log.Printf("view: rendering %d page: %v", status, detail)
	}
	if !lang.Valid() {
		lang = i18n.Default()
	}
	titleKey, messageKey := "error.500.title", "error.500.message"
	if status == http.StatusNotFound {
		titleKey, messageKey = "error.404.title", "error.404.message"
	}

	// The nav is built from fixed section paths, so an error page needs no
	// database access — the last thing an error path should do is query.
	base := "/" + lang.String()
	data := viewmodel.ErrorViewModel{
		Meta: viewmodel.Meta{
			SiteName:    r.dict.Message(lang, "site.name"),
			Title:       r.dict.Message(lang, titleKey),
			Description: r.dict.Message(lang, messageKey),
			Lang:        lang,
			HTMLLang:    lang.HTMLLang(),
			NoIndex:     true,
		},
		Nav:     viewmodel.NewNav(lang, ""),
		Status:  status,
		Title:   r.dict.Message(lang, titleKey),
		Message: r.dict.Message(lang, messageKey),
		HomeURL: base,
	}
	if err := r.write(w, data, ErrorView, status); err != nil {
		log.Printf("view: rendering %d error page: %v", status, err)
		http.Error(w, http.StatusText(status), status)
	}
}

func (r *Renderer) write(w http.ResponseWriter, data any, name string, status int) error {
	meta := viewmodel.MetaOf(data)
	byView, ok := r.sets[meta.Lang]
	if !ok {
		return fmt.Errorf("view: no template sets for language %q", meta.Lang)
	}
	t, ok := byView[name]
	if !ok {
		return fmt.Errorf("view: unknown view %q", name)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout.html", document{Meta: meta, Data: data}); err != nil {
		return fmt.Errorf("view: execute %q (%s): %w", name, meta.Lang, err)
	}

	if status != http.StatusOK {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}

// document is the single value every template set is executed with. The layout
// reads .Meta for <head> and every partial reads .Data.
type document struct {
	Meta viewmodel.Meta
	Data any
}

func statusOf(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}

// NotFoundHandler serves the 404 page, except under /api, where the reserved
// JSON API must keep answering with net/http's plain-text 404 exactly as before.
func (r *Renderer) NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/api" || strings.HasPrefix(req.URL.Path, "/api/") {
			http.NotFound(w, req)
			return
		}
		lang, _ := i18n.FromPath(req.URL.Path)
		r.RenderError(w, lang, http.StatusNotFound, nil)
	}
}

// Handler adapts a controller that returns a Page to an http.HandlerFunc.
//
// The controller stays free of the renderer, so the two can be tested apart. It
// is deliberately not chi middleware: middleware cannot see a handler's return
// value, which is where the page lives.
func (r *Renderer) Handler(controller func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return controller
}
