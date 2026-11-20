// Package viewmodel holds presentation-shaped data for the public pages.
//
// It follows the shape the reference project uses: one type per page
// (HomeViewModel, ArticleListViewModel, ArticleViewModel, GalleryViewModel, …),
// each embedding Meta for the <head>. Types carry no HTTP wiring and no
// interface wording — the site is Chinese-only, so static copy lives in the
// templates — and the templates only print fields.
package viewmodel

import (
	"html/template"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// MetaOf returns the head metadata of any page view model. Every page view model
// embeds Meta, so the renderer needs no per-view knowledge.
func MetaOf(data any) Meta {
	switch v := data.(type) {
	case HomeViewModel:
		return v.Meta
	case ArticleListViewModel:
		return v.Meta
	case ArticleViewModel:
		return v.Meta
	case GalleryViewModel:
		return v.Meta
	case AboutViewModel:
		return v.Meta
	case ErrorViewModel:
		return v.Meta
	default:
		return Meta{}
	}
}

// Meta is the per-page <head> data.
type Meta struct {
	SiteName    string
	Title       string
	Description string
	Path        string
}

// Active reports whether the current path is inside the given section. An empty
// segment means the home section, which owns the root only.
func (m Meta) Active(segment string) bool { return sectionActive(segment, m.Path) }

// Link is a navigational target. Label is data — a category name, a tag name, a
// page number — never interface wording.
type Link struct {
	Label  string
	Href   string
	Active bool
}

// HomePath is the canonical address of the site landing page.
func HomePath() string { return "/" }

// SectionPath is the canonical address of one top-level section.
func SectionPath(segment string) string {
	if segment == "" {
		return HomePath()
	}
	return "/" + segment
}

// sectionActive reports whether path is inside the given section. An empty path
// (the error page) marks nothing active. The home section owns the root only.
func sectionActive(segment, path string) bool {
	if path == "" {
		return false
	}
	rest := strings.Trim(path, "/")
	if segment == "" {
		return rest == ""
	}
	return rest == segment || strings.HasPrefix(rest, segment+"/")
}

// Pagination is a page window plus the links that move through it.
type Pagination struct {
	Page       int
	TotalPages int
	Total      int64
	Prev       *Link
	Next       *Link
	First      *Link
	Last       *Link
	Pages      []Link
}

// NewPagination builds the pagination of a list page. base is the canonical
// address of the page and query carries the active filters, so every link
// preserves them.
func NewPagination(base string, query ListQuery, page, totalPages int, total int64) *Pagination {
	if totalPages <= 0 {
		return nil
	}
	href := func(n int) string { return query.URL(base, n) }

	p := &Pagination{Page: page, TotalPages: totalPages, Total: total}
	if page > 1 {
		p.Prev = &Link{Href: href(page - 1)}
		p.First = &Link{Href: href(1)}
	}
	if page < totalPages {
		p.Next = &Link{Href: href(page + 1)}
		p.Last = &Link{Href: href(totalPages)}
	}

	// A window around the current page keeps the control readable at any total.
	const window = 2
	from, to := page-window, page+window
	if from < 1 {
		from = 1
	}
	if to > totalPages {
		to = totalPages
	}
	for n := from; n <= to; n++ {
		p.Pages = append(p.Pages, Link{Href: href(n), Label: strconv.Itoa(n), Active: n == page})
	}
	return p
}

// FormatDate renders an instant for display and for a <time datetime> attribute.
// The display form is Chinese; the attribute is RFC 3339 UTC so the value is
// unambiguous to machines.
func FormatDate(t time.Time, zone *time.Location) (iso, display string) {
	if t.IsZero() {
		return "", ""
	}
	if zone != nil {
		t = t.In(zone)
	}
	iso = t.UTC().Format(time.RFC3339)
	display = t.Format("2006年01月02日")
	return iso, display
}

// FormatMonth renders a month heading for the list page.
func FormatMonth(year int, month time.Month) string {
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006年01月")
}

// Truncate shortens s to at most max runes without splitting a character. It is
// used for meta descriptions, where a Chinese summary would otherwise be cut
// mid-word by any byte-oriented limit.
func Truncate(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return strings.TrimRight(string(runes[:max]), " ") + "…"
}

// Describe builds the <meta name="description"> for a page: the given text, or
// the fallback when there is none.
func Describe(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	return Truncate(text, 160)
}

// TrustedHTML marks a string as already-rendered HTML. It is the single escape
// hatch in the view layer.
//
// The only values that may pass through it are article bodies, which the admin
// application renders from Markdown at save time. Never call it with request
// input.
func TrustedHTML(s string) template.HTML { return template.HTML(s) }

// IndexOf zero-pads a list position the way the article rows show it: [01].
func IndexOf(i int) string {
	n := i + 1
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
