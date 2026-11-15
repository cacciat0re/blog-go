package handler

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"gitlab.arorms.cn/arorms/applications/blog/internal/i18n"
	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// maxPage bounds the page number so (page-1)*limit cannot overflow an int on a
// crawled or hand-edited URL. It is far beyond any real archive.
const maxPage = 100_000

// maxKeyRunes bounds the search term before it reaches an ILIKE pattern.
const maxKeyRunes = 100

// listParams is the validated input of a list page.
type listParams struct {
	Lang     i18n.Lang
	Page     int
	Limit    int
	Key      string
	Category string
	Tag      string
}

// parseListParams reads the language from the path and the page state from the
// query. Anything malformed is coerced to a sane default rather than rejected:
// a crawlable page should still render, and page 1 is the honest answer for an
// unreadable page number.
func parseListParams(r *http.Request) listParams {
	lang, ok := i18n.FromPath(r.URL.Path)
	if !ok {
		lang = i18n.Default()
	}

	query := r.URL.Query()
	p := listParams{
		Lang:     lang,
		Page:     parsePage(query.Get("page")),
		Limit:    parseLimit(query.Get("limit")),
		Key:      trimTo(query.Get("key"), maxKeyRunes),
		Category: strings.TrimSpace(query.Get("category")),
		Tag:      strings.TrimSpace(query.Get("tag")),
	}
	return p
}

// Payload converts to the shape internal/service consumes through the view-layer
// query, so the handler has exactly one definition of what a page means.
func (p listParams) Payload() payload.ArticleListRequest { return p.Query().Payload() }

// Query converts to the view-layer query used to build addresses. Limit is
// already clamped, so the offset and the pagination links always agree.
func (p listParams) Query() viewmodel.ListQuery {
	return viewmodel.ListQuery{
		Lang:     p.Lang.String(),
		Key:      p.Key,
		Category: p.Category,
		Tag:      p.Tag,
		Page:     p.Page,
		Limit:    p.Limit,
	}
}

// Filters reports whether any filter is active, which decides whether search
// results stay out of the index.
func (p listParams) Filters() bool {
	return p.Key != "" || p.Category != "" || p.Tag != ""
}

func parsePage(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 {
		return 1
	}
	if n > maxPage {
		return maxPage
	}
	return n
}

func parseLimit(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return viewmodel.DefaultListLimit
	}
	if n > viewmodel.MaxListLimit {
		return viewmodel.MaxListLimit
	}
	return n
}

func trimTo(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
