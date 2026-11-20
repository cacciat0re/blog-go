package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/service"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// maxPage bounds the page number so (page-1)*limit cannot overflow an int on a
// crawled or hand-edited URL. It is far beyond any real listing.
const maxPage = 100_000

// maxKeyRunes bounds the search term before it reaches an ILIKE pattern.
const maxKeyRunes = 100

// ArticlesHandler serves the article list and the article detail page.
type ArticlesHandler struct {
	articles   service.IArticleService
	categories service.ICategoryService
	tags       service.ITagService
}

func NewArticlesHandler(articles service.IArticleService, categories service.ICategoryService, tags service.ITagService) *ArticlesHandler {
	return &ArticlesHandler{articles: articles, categories: categories, tags: tags}
}

// List renders the paginated article list.
func (h *ArticlesHandler) List(w http.ResponseWriter, r *http.Request) {
	serve(w, r, func() (view.Page, error) {
		p := parseListParams(r)

		// A filter naming something that does not exist is a 404. An empty list
		// would let a crawler mint unlimited empty URLs.
		if !h.categoryExists(r.Context(), p.Category) {
			return view.Page{}, view.NotFound(nil)
		}
		if !h.tagExists(r.Context(), p.Tag) {
			return view.Page{}, view.NotFound(nil)
		}

		result, err := h.articles.GetAllPublished(r.Context(), p.Payload())
		if err != nil {
			return view.Page{}, err
		}
		// Page numbers past the end are a 404 for the same reason; page 1 of an
		// empty list is a legitimate page and stays a 200.
		if p.Page > 1 && (result.Total == 0 || p.Page > result.TotalPages) {
			return view.Page{}, view.NotFound(nil)
		}

		filters, err := h.filters(r.Context(), p)
		if err != nil {
			return view.Page{}, err
		}

		base := viewmodel.SectionPath("articles")
		return view.Page{
			View: view.ArticleListView,
			Data: viewmodel.ArticleListViewModel{
				Meta:       meta(r.URL.Path, "文章", view.SiteDescription),
				BasePath:   base,
				ClearHref:  base,
				Query:      p.Query(),
				Groups:     h.groupByMonth(result.Content, base),
				Shown:      len(result.Content),
				Total:      result.Total,
				TotalPages: result.TotalPages,
				Filters:    filters,
				Pagination: viewmodel.NewPagination(base, p.Query(), p.Page, result.TotalPages, result.Total),
				HasFilters: p.Filters(),
				Empty:      len(result.Content) == 0,
			},
		}, nil
	})
}

// Detail renders one article.
func (h *ArticlesHandler) Detail(w http.ResponseWriter, r *http.Request) {
	serve(w, r, func() (view.Page, error) {
		slug := r.PathValue("slug")
		if slug == "" {
			return view.Page{}, view.NotFound(nil)
		}

		article, err := h.articles.GetPublishedDetail(r.Context(), slug)
		if err != nil {
			if errors.Is(err, service.ErrArticleNotFound) {
				return view.Page{}, view.NotFound(err)
			}
			return view.Page{}, err
		}

		createdISO, createdDisplay := viewmodel.FormatDate(article.CreatedAt, zone)
		updatedISO, updatedDisplay := viewmodel.FormatDate(article.UpdatedAt, zone)
		articlesPath := viewmodel.SectionPath("articles")

		data := viewmodel.ArticleViewModel{
			Meta:           meta(r.URL.Path, article.Title, article.Summary),
			ID:             article.ID,
			Slug:           article.Slug,
			Title:          article.Title,
			Summary:        article.Summary,
			Content:        viewmodel.TrustedHTML(article.Content),
			DateISO:        createdISO,
			DateDisplay:    createdDisplay,
			UpdatedISO:     updatedISO,
			UpdatedDisplay: updatedDisplay,
			BackHref:       articlesPath,
		}
		if article.Category != nil {
			data.Category = &viewmodel.Link{
				Label: article.Category.Name,
				Href:  articlesPath + "?category=" + article.Category.Slug,
			}
		}
		for _, tag := range article.Tags {
			data.Tags = append(data.Tags, viewmodel.Link{
				Label: tag.Name,
				Href:  articlesPath + "?tag=" + tag.Slug,
			})
		}

		return view.Page{View: view.ArticleDetailView, Data: data}, nil
	})
}

// groupByMonth groups rows the way the legacy frontend did: newest month first,
// in the order the service already returned them.
func (h *ArticlesHandler) groupByMonth(items []payload.ArticleListItemResponse, base string) []viewmodel.MonthGroupViewModel {
	groups := make([]viewmodel.MonthGroupViewModel, 0, len(items))
	index := make(map[string]int, len(items))
	for i, item := range items {
		at := item.CreatedAt
		if zone != nil {
			at = at.In(zone)
		}
		key := at.Format("2006-01")
		groupIndex, ok := index[key]
		if !ok {
			groupIndex = len(groups)
			index[key] = groupIndex
			groups = append(groups, viewmodel.MonthGroupViewModel{
				Label: viewmodel.FormatMonth(at.Year(), at.Month()),
			})
		}
		groups[groupIndex].Items = append(groups[groupIndex].Items, summary(item, i, base))
	}
	return groups
}

// filters builds the category chips of the list page.
func (h *ArticlesHandler) filters(ctx context.Context, p listParams) ([]viewmodel.Link, error) {
	categories, err := h.categories.List(ctx)
	if err != nil {
		return nil, err
	}

	base := viewmodel.SectionPath("articles")
	query := p.Query()

	out := make([]viewmodel.Link, 0, len(categories)+1)
	out = append(out, viewmodel.Link{
		Label:  "全部",
		Href:   query.Without("category").URL(base, 1),
		Active: p.Category == "",
	})
	for _, c := range categories {
		out = append(out, viewmodel.Link{
			Label:  c.Name,
			Href:   query.WithCategory(c.Slug).URL(base, 1),
			Active: p.Category == c.Slug,
		})
	}
	return out, nil
}

func (h *ArticlesHandler) categoryExists(ctx context.Context, slug string) bool {
	if slug == "" {
		return true
	}
	_, err := h.categories.GetBySlug(ctx, slug)
	return err == nil
}

func (h *ArticlesHandler) tagExists(ctx context.Context, slug string) bool {
	if slug == "" {
		return true
	}
	_, err := h.tags.GetBySlug(ctx, slug)
	return err == nil
}

// listParams is the validated input of the article list: the page state is read
// from the query string, which is what an MVC page exposes. The JSON API parses
// its own request and does not share this type.
type listParams struct {
	Page     int
	Limit    int
	Key      string
	Category string
	Tag      string
}

// parseListParams reads the page state from the query. Anything malformed is
// coerced to a sane default rather than rejected: a crawlable page should still
// render, and page 1 is the honest answer for an unreadable page number.
func parseListParams(r *http.Request) listParams {
	query := r.URL.Query()
	return listParams{
		Page:     parsePage(query.Get("page")),
		Limit:    parseLimit(query.Get("limit")),
		Key:      trimTo(query.Get("key"), maxKeyRunes),
		Category: strings.TrimSpace(query.Get("category")),
		Tag:      strings.TrimSpace(query.Get("tag")),
	}
}

// Payload converts to the shape internal/service consumes through the view-layer
// query, so the page has exactly one definition of what a page means.
func (p listParams) Payload() payload.ArticleListRequest { return p.Query().Payload() }

// Query converts to the view-layer query used to build addresses. Limit is
// already clamped, so the offset and the pagination links always agree.
func (p listParams) Query() viewmodel.ListQuery {
	return viewmodel.ListQuery{
		Key:      p.Key,
		Category: p.Category,
		Tag:      p.Tag,
		Page:     p.Page,
		Limit:    p.Limit,
	}
}

// Filters reports whether any filter is active.
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

// trimTo trims s and shortens it to max runes without splitting a character.
func trimTo(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
