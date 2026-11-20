package viewmodel

import (
	"html/template"
)

// ArticleSummaryViewModel is one row of an article list. It mirrors the legacy
// ArticleSummaryVo: identity, title, summary, category, tags, plus the
// preformatted dates the row prints.
type ArticleSummaryViewModel struct {
	ID          int64
	Slug        string
	Title       string
	Summary     string
	Href        string
	Index       string // zero-padded list position, e.g. "01"
	Category    *Link
	Tags        []Link
	DateISO     string
	DateDisplay string
}

// HomeViewModel is the landing page: the latest articles.
type HomeViewModel struct {
	Meta
	Articles []ArticleSummaryViewModel
}

// MonthGroupViewModel is one heading of the list page.
type MonthGroupViewModel struct {
	Label string
	Items []ArticleSummaryViewModel
}

// ArticleListViewModel is the paginated list of published articles.
type ArticleListViewModel struct {
	Meta
	BasePath   string
	ClearHref  string
	Query      ListQuery
	Groups     []MonthGroupViewModel
	Shown      int
	Total      int64
	TotalPages int
	Filters    []Link
	Pagination *Pagination
	HasFilters bool
	Empty      bool
}

// ArticleViewModel is one article. It mirrors the legacy ArticleVo.
type ArticleViewModel struct {
	Meta
	ID             int64
	Slug           string
	Title          string
	Summary        string
	Content        template.HTML
	Category       *Link
	Tags           []Link
	DateISO        string
	DateDisplay    string
	UpdatedISO     string
	UpdatedDisplay string
	BackHref       string
}

// GalleryViewModel is the picture gallery.
type GalleryViewModel struct {
	Meta
	Pictures   []PictureViewModel
	Total      int64
	Pagination *Pagination
	Empty      bool
}

// PictureViewModel is one gallery frame.
type PictureViewModel struct {
	ID        int64
	Alt       string
	URL       string
	Thumbnail string
	DateISO   string
	DateShow  string
	Tags      []Link
}

// AboutViewModel is the static about page; its copy lives in the template.
type AboutViewModel struct {
	Meta
}

// ErrorViewModel is the shared error page.
type ErrorViewModel struct {
	Meta
	Status  int
	Title   string
	Message string
	HomeURL string
}
