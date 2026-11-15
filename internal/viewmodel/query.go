package viewmodel

import (
	"net/url"
	"strconv"
	"strings"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
)

// Page limits mirrored from internal/payload so the view layer stays free of
// that package's request shapes.
const (
	DefaultListLimit = 20
	MaxListLimit     = 100
)

// ListQuery is the validated query of a list page: everything here is already
// normalised (trimmed, clamped, language checked).
type ListQuery struct {
	Lang     string // URL code, e.g. "en"
	Key      string
	Category string
	Tag      string
	Page     int
	Limit    int
}

// Offset is the row offset for page.
func (q ListQuery) Offset() int {
	page := q.Page
	if page < 1 {
		page = 1
	}
	return (page - 1) * q.Limit
}

// URL builds the canonical address of a list page, keeping the active filters.
// The default page and the default limit are omitted so one page has one URL.
func (q ListQuery) URL(base string, page int) string {
	values := url.Values{}
	if q.Key != "" {
		values.Set("key", q.Key)
	}
	if q.Category != "" {
		values.Set("category", q.Category)
	}
	if q.Tag != "" {
		values.Set("tag", q.Tag)
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if q.Limit > 0 && q.Limit != DefaultListLimit {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	if len(values) == 0 {
		return base
	}
	return base + "?" + values.Encode()
}

// HomeURL is the list address at its first page with no pagination suffix.
func (q ListQuery) HomeURL(base string) string { return q.URL(base, 1) }

// Payload converts to the shape internal/service consumes. Limit must already be
// clamped: the offset is derived from it.
func (q ListQuery) Payload() payload.ArticleListRequest {
	return payload.ArticleListRequest{
		Language: strings.ToUpper(q.Lang),
		Key:      q.Key,
		Category: q.Category,
		Tag:      q.Tag,
		Limit:    q.Limit,
		Offset:   q.Offset(),
	}
}

// WithCategory returns the same query filtered to one category, on page 1.
func (q ListQuery) WithCategory(slug string) ListQuery {
	q.Category, q.Page = slug, 1
	return q
}

// WithTag returns the same query filtered to one tag, on page 1.
func (q ListQuery) WithTag(slug string) ListQuery {
	q.Tag, q.Page = slug, 1
	return q
}

// Without returns the query with one dimension of the filter cleared, on page 1.
// The other filter is kept: a reader narrowing by tag should be able to change
// the category without losing the tag.
func (q ListQuery) Without(dimension string) ListQuery {
	switch dimension {
	case "category":
		q.Category = ""
	case "tag":
		q.Tag = ""
	case "key":
		q.Key = ""
	}
	q.Page = 1
	return q
}
