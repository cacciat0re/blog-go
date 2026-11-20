// Package view renders the public HTML pages with the standard library's
// html/template.
//
// The site is Chinese-only: there is one template set, parsed at boot.
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

	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// View names identify a template set. They are logical names, not filenames: a
// handler never needs to know which .html file ends up being used.
const (
	HomeView          = "home"
	ArticleListView   = "article-list"
	ArticleDetailView = "article-detail"
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

// The embedded file set: the source templates plus the assets that are served.
//
// The CSS is listed file by file rather than as "assets" so the Tailwind source
// (assets/css/input.css) stays out of the binary — it is a build input, not
// something a browser should ever be able to download.
//
//go:embed templates
//go:embed assets/css/index.css assets/css/katex
//go:embed assets/img
var files embed.FS

// pageFiles maps a view name to the file that defines its "content" block.
var pageFiles = map[string]string{
	HomeView:          "home.html",
	ArticleListView:   "article-list.html",
	ArticleDetailView: "article-detail.html",
	GalleryView:       "gallery.html",
	AboutView:         "about.html",
	ErrorView:         "error.html",
}

// Renderer holds one parsed template set per view.
type Renderer struct {
	sets map[string]*template.Template
}

// New parses every template set.
//
// Parsing is eager because a template error should stop the process at boot
// instead of failing a reader's page later.
func New() (*Renderer, error) {
	r := &Renderer{sets: make(map[string]*template.Template, len(pageFiles))}
	for name, page := range pageFiles {
		t, err := template.New("layout.html").ParseFS(files,
			"templates/layouts/layout.html",
			"templates/partials/*.html",
			"templates/"+page,
		)
		if err != nil {
			return nil, fmt.Errorf("parse view %q: %w", name, err)
		}
		if t.Lookup("layout.html") == nil {
			return nil, fmt.Errorf("view %q: layout template missing", name)
		}
		if t.Lookup("content") == nil {
			return nil, fmt.Errorf("view %q: %s does not define \"content\"", name, page)
		}
		r.sets[name] = t
	}
	return r, nil
}

// Assets returns the embedded static files, addressed as the templates refer to
// them: /assets/index.css, /assets/katex/…, /assets/img/….
func Assets() fs.FS { return assetFS{files: files} }

// Render writes a page.
//
// The document is rendered into a buffer first: if template execution fails,
// nothing has been written yet and the caller can still answer with a clean 500.
func (r *Renderer) Render(w http.ResponseWriter, p Page) error {
	return r.write(w, p.Data, p.View, statusOf(p.Status))
}

// RenderError writes the shared error page for status. It never fails silently:
// when even the error page cannot be rendered, it falls back to a plain-text
// status body.
func (r *Renderer) RenderError(w http.ResponseWriter, status int, detail error) {
	if status >= http.StatusInternalServerError && detail != nil {
		log.Printf("view: rendering %d page: %v", status, detail)
	}

	title, message := errorText(status)

	// The nav is built from fixed section paths, so an error page needs no
	// database access — the last thing an error path should do is query.
	data := viewmodel.ErrorViewModel{
		Meta: viewmodel.Meta{
			SiteName:    SiteName,
			Title:       title,
			Description: message,
		},
		Status:  status,
		Title:   title,
		Message: message,
		HomeURL: "/",
	}
	if err := r.write(w, data, ErrorView, status); err != nil {
		log.Printf("view: rendering %d error page: %v", status, err)
		http.Error(w, http.StatusText(status), status)
	}
}

func (r *Renderer) write(w http.ResponseWriter, data any, name string, status int) error {
	meta := viewmodel.MetaOf(data)

	t, ok := r.sets[name]
	if !ok {
		return fmt.Errorf("view: unknown view %q", name)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout.html", document{Meta: meta, Data: data}); err != nil {
		return fmt.Errorf("view: execute %q: %w", name, err)
	}

	w.Header().Set("Content-Language", "zh-CN")
	if status != http.StatusOK {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "private, no-cache")
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

// errorText is the fixed copy of the shared error page.
func errorText(status int) (title, message string) {
	if status == http.StatusNotFound {
		return "页面未找到", "你访问的地址在本站不存在。"
	}
	return "服务器出错", "服务器无法生成这个页面，错误已经记录。"
}
