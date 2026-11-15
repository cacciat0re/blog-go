package viewmodel

import (
	"html/template"
)

// ArticleSummaryViewModel is one row of an article list. It mirrors the legacy
// ArticleSummaryVo: identity, title, summary, category, tags, plus the
// preformatted dates the row prints.
type ArticleSummaryViewModel struct {
	ID               int64
	Slug             string
	Title            string
	Summary          string
	Href             string
	Index            string // zero-padded list position, e.g. "01"
	Category         *Link
	Tags             []Link
	DateISO          string
	DateDisplay      string
	IsTranslatedByAI bool
}

// HomeViewModel is the landing page: the latest articles and the site counters.
type HomeViewModel struct {
	SharedViewModel
	Articles     []ArticleSummaryViewModel
	ArticlesHref string
	ArchiveHref  string
	Stats        []StatViewModel
}

// StatViewModel is one counter of the ruler under the masthead.
type StatViewModel struct {
	Value string
	Label string // an i18n key, translated in the template
}

// MonthGroupViewModel is one heading of the list page.
type MonthGroupViewModel struct {
	Label string
	Items []ArticleSummaryViewModel
}

// ArticleListViewModel is the paginated archive of published articles.
type ArticleListViewModel struct {
	SharedViewModel
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
	SharedViewModel
	ID               int64
	Slug             string
	Title            string
	Summary          string
	Content          template.HTML
	Category         *Link
	Tags             []Link
	DateISO          string
	DateDisplay      string
	UpdatedISO       string
	UpdatedDisplay   string
	IsTranslatedByAI bool
	BackHref         string
}

// ArchiveViewModel is the category index, nested one level.
type ArchiveViewModel struct {
	SharedViewModel
	Groups []CategoryGroupViewModel
	Empty  bool
}

// CategoryGroupViewModel is a top-level category with its children.
type CategoryGroupViewModel struct {
	ID       int64
	Name     string
	Href     string
	Children []Link
}

// GalleryViewModel is the picture gallery.
type GalleryViewModel struct {
	SharedViewModel
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

// AboutViewModel is the static about page; its copy lives in the locale files.
type AboutViewModel struct {
	SharedViewModel
}

// ErrorViewModel is the shared error page.
type ErrorViewModel struct {
	SharedViewModel
	Status  int
	Title   string
	Message string
	HomeURL string
}
