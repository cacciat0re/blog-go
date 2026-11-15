package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"gitlab.arorms.cn/arorms/applications/blog/internal/i18n"
	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/service"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
	"gitlab.arorms.cn/arorms/applications/blog/internal/viewmodel"
)

// homeLatest is how many articles the landing page shows, matching the legacy
// homepage which sliced the first four rows of a six-row query.
const homeLatest = 4

// homeQuerySize is how many rows the landing page asks for; the legacy client
// requested six so it could show four plus its own headings.
const homeQuerySize = 6

// PageHandler serves the public HTML pages.
//
// Every method is a controller: it validates the request, asks the services for
// data, and returns a view.Page. It never executes a template and never writes a
// response body — both are the renderer's job.
type PageHandler struct {
	articles   service.IArticleService
	categories service.ICategoryService
	tags       service.ITagService
	pictures   service.IPictureService
	stats      service.IStatisticsService
	renderer   *view.Renderer
	dict       *i18n.Dictionary
	zone       *time.Location
}

// NewPageHandler wires the public pages. zone is the timezone dates are shown
// in; nil means UTC.
func NewPageHandler(
	articles service.IArticleService,
	categories service.ICategoryService,
	tags service.ITagService,
	pictures service.IPictureService,
	stats service.IStatisticsService,
	renderer *view.Renderer,
	dict *i18n.Dictionary,
	zone *time.Location,
) *PageHandler {
	return &PageHandler{
		articles:   articles,
		categories: categories,
		tags:       tags,
		pictures:   pictures,
		stats:      stats,
		renderer:   renderer,
		dict:       dict,
		zone:       zone,
	}
}

// Home renders the landing page: masthead, counters, latest articles.
func (h *PageHandler) Home(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, func() (view.Page, error) {
		lang := pageLang(r)
		query := viewmodel.ListQuery{Lang: lang.String(), Page: 1, Limit: homeQuerySize}

		result, err := h.articles.GetAllPublished(r.Context(), query.Payload())
		if err != nil {
			return view.Page{}, err
		}
		rows := result.Content
		if len(rows) > homeLatest {
			rows = rows[:homeLatest]
		}

		// Rows link into the articles section, not the language root.
		base := viewmodel.SectionPath(lang, "articles")
		return view.Page{
			View: view.HomeView,
			Data: viewmodel.HomeViewModel{
				SharedViewModel: viewmodel.SharedViewModel{
					Meta: h.rootMeta(lang, r.URL.Path),
					Nav:  viewmodel.NewNav(lang, r.URL.Path),
				},
				Articles:     h.summaries(rows, lang, base),
				ArticlesHref: viewmodel.SectionPath(lang, "articles"),
				ArchiveHref:  viewmodel.SectionPath(lang, "archive"),
				Stats:        h.statsView(r.Context(), lang),
			},
		}, nil
	})
}

// List renders the paginated article list.
func (h *PageHandler) List(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, func() (view.Page, error) {
		p := parseListParams(r)

		// A filter naming something that does not exist is a 404. An empty list
		// would let a crawler mint unlimited empty URLs.
		if !h.categoryExists(r.Context(), p.Category, p.Lang) {
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
		// empty archive is a legitimate page and stays a 200.
		if p.Page > 1 && (result.Total == 0 || p.Page > result.TotalPages) {
			return view.Page{}, view.NotFound(nil)
		}

		filters, err := h.filters(r.Context(), p)
		if err != nil {
			return view.Page{}, err
		}

		base := viewmodel.SectionPath(p.Lang, "articles")
		return view.Page{
			View: view.ArticleListView,
			Data: viewmodel.ArticleListViewModel{
				SharedViewModel: viewmodel.SharedViewModel{
					Meta: h.meta(p.Lang, r.URL.Path, "articles.title", "site.description"),
					Nav:  viewmodel.NewNav(p.Lang, r.URL.Path),
				},
				BasePath:   base,
				ClearHref:  base,
				Query:      p.Query(),
				Groups:     h.groupByMonth(result.Content, p.Lang, base),
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
func (h *PageHandler) Detail(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, func() (view.Page, error) {
		lang := pageLang(r)
		slug := r.PathValue("slug")
		if slug == "" {
			return view.Page{}, view.NotFound(nil)
		}

		article, err := h.articles.GetPublishedDetail(r.Context(), slug, lang.Payload())
		if err != nil {
			if errors.Is(err, service.ErrArticleNotFound) {
				return view.Page{}, view.NotFound(err)
			}
			return view.Page{}, err
		}

		createdISO, createdDisplay := viewmodel.FormatDate(article.CreatedAt, lang, h.zone)
		updatedISO, updatedDisplay := viewmodel.FormatDate(article.UpdatedAt, lang, h.zone)
		articlesPath := viewmodel.SectionPath(lang, "articles")

		data := viewmodel.ArticleViewModel{
			SharedViewModel: viewmodel.SharedViewModel{
				Meta: h.textMeta(lang, r.URL.Path, article.Title, article.Summary),
				Nav:  viewmodel.NewNav(lang, r.URL.Path),
			},
			ID:               article.ID,
			Slug:             article.Slug,
			Title:            article.Title,
			Summary:          article.Summary,
			Content:          viewmodel.TrustedHTML(article.Content),
			DateISO:          createdISO,
			DateDisplay:      createdDisplay,
			UpdatedISO:       updatedISO,
			UpdatedDisplay:   updatedDisplay,
			IsTranslatedByAI: article.IsTranslatedByAI,
			BackHref:         articlesPath,
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

// Archive renders the category index.
func (h *PageHandler) Archive(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, func() (view.Page, error) {
		lang := pageLang(r)

		tree, err := h.categories.Tree(r.Context(), lang.Payload())
		if err != nil {
			return view.Page{}, err
		}

		articlesPath := viewmodel.SectionPath(lang, "articles")
		groups := make([]viewmodel.CategoryGroupViewModel, 0, len(tree))
		for _, node := range tree {
			group := viewmodel.CategoryGroupViewModel{
				ID:   node.ID,
				Name: node.Name,
				Href: articlesPath + "?category=" + node.Slug,
			}
			for _, child := range node.Children {
				group.Children = append(group.Children, viewmodel.Link{
					Label: child.Name,
					Href:  articlesPath + "?category=" + child.Slug,
				})
			}
			groups = append(groups, group)
		}

		return view.Page{
			View: view.ArchiveView,
			Data: viewmodel.ArchiveViewModel{
				SharedViewModel: viewmodel.SharedViewModel{
					Meta: h.meta(lang, r.URL.Path, "archive.title", "archive.intro"),
					Nav:  viewmodel.NewNav(lang, r.URL.Path),
				},
				Groups: groups,
				Empty:  len(groups) == 0,
			},
		}, nil
	})
}

// Gallery renders the picture gallery.
func (h *PageHandler) Gallery(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, func() (view.Page, error) {
		lang := pageLang(r)
		p := parseListParams(r)

		// The gallery is not paginated by the article list's filters, only by page.
		result, err := h.pictures.GetGallery(r.Context(), p.Limit, (p.Page-1)*p.Limit)
		if err != nil {
			return view.Page{}, err
		}
		if p.Page > 1 && (result.Total == 0 || p.Page > result.TotalPages) {
			return view.Page{}, view.NotFound(nil)
		}

		base := viewmodel.SectionPath(lang, "gallery")
		query := viewmodel.ListQuery{Lang: lang.String(), Page: p.Page, Limit: p.Limit}

		pictures := make([]viewmodel.PictureViewModel, 0, len(result.Content))
		for _, item := range result.Content {
			createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
			if err != nil {
				return view.Page{}, fmt.Errorf("parse picture %d timestamp: %w", item.ID, err)
			}
			iso, display := viewmodel.FormatDate(createdAt, lang, h.zone)
			frame := viewmodel.PictureViewModel{
				ID:        item.ID,
				Alt:       item.Alt,
				URL:       item.URL,
				Thumbnail: item.ThumbnailURL,
				DateISO:   iso,
				DateShow:  display,
			}
			for _, tag := range item.Tags {
				frame.Tags = append(frame.Tags, viewmodel.Link{
					Label: tag.Name,
					Href:  viewmodel.SectionPath(lang, "articles") + "?tag=" + tag.Slug,
				})
			}
			pictures = append(pictures, frame)
		}

		// An empty gallery is not worth indexing; a populated one is a normal page.
		meta := h.meta(lang, r.URL.Path, "gallery.title", "gallery.intro")
		meta.NoIndex = len(pictures) == 0

		return view.Page{
			View: view.GalleryView,
			Data: viewmodel.GalleryViewModel{
				SharedViewModel: viewmodel.SharedViewModel{
					Meta: meta,
					Nav:  viewmodel.NewNav(lang, r.URL.Path),
				},
				Pictures:   pictures,
				Total:      result.Total,
				Pagination: viewmodel.NewPagination(base, query, p.Page, result.TotalPages, result.Total),
				Empty:      len(pictures) == 0,
			},
		}, nil
	})
}

// About renders the static about page.
func (h *PageHandler) About(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, func() (view.Page, error) {
		lang := pageLang(r)
		return view.Page{
			View: view.AboutView,
			Data: viewmodel.AboutViewModel{
				SharedViewModel: viewmodel.SharedViewModel{
					Meta: h.meta(lang, r.URL.Path, "about.title", "site.description"),
					Nav:  viewmodel.NewNav(lang, r.URL.Path),
				},
			},
		}, nil
	})
}

// serve is the single adapter between a controller and the renderer. It is the
// only place that maps an outcome to a response, so no controller repeats it and
// none of them writes a body.
func (h *PageHandler) serve(w http.ResponseWriter, r *http.Request, controller func() (view.Page, error)) {
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
		h.renderer.RenderError(w, pageLang(r), status, err)
		return
	}

	if err := h.renderer.Render(w, page); err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		h.renderer.RenderError(w, pageLang(r), http.StatusInternalServerError, err)
	}
}

// meta builds head data from two locale keys: the page title and its
// description.
func (h *PageHandler) meta(lang i18n.Lang, path, titleKey, descriptionKey string) viewmodel.Meta {
	return h.textMeta(lang, path, h.dict.Message(lang, titleKey), h.dict.Message(lang, descriptionKey))
}

// rootMeta is the landing page's head: its title is the site itself, so it is
// not wrapped in the site-title pattern.
func (h *PageHandler) rootMeta(lang i18n.Lang, path string) viewmodel.Meta {
	meta := h.meta(lang, path, "site.name", "site.description")
	meta.Title = h.dict.Message(lang, "site.name")
	return meta
}

// textMeta applies the site's title pattern to literal page text, which is what
// an article page needs.
func (h *PageHandler) textMeta(lang i18n.Lang, path, title, description string) viewmodel.Meta {
	return viewmodel.Meta{
		SiteName:    h.dict.Message(lang, "site.name"),
		Title:       fmt.Sprintf(h.dict.Message(lang, "meta.titleFormat"), title),
		Description: viewmodel.Describe(description, h.dict.Message(lang, "site.description")),
		Lang:        lang,
		HTMLLang:    lang.HTMLLang(),
		Path:        path,
	}
}

// sums converts list rows into view rows.
func (h *PageHandler) summaries(items []payload.ArticleListItemResponse, lang i18n.Lang, base string) []viewmodel.ArticleSummaryViewModel {
	out := make([]viewmodel.ArticleSummaryViewModel, 0, len(items))
	for i, item := range items {
		out = append(out, h.summary(item, i, lang, base))
	}
	return out
}

func (h *PageHandler) summary(item payload.ArticleListItemResponse, index int, lang i18n.Lang, base string) viewmodel.ArticleSummaryViewModel {
	iso, display := viewmodel.FormatDate(item.CreatedAt, lang, h.zone)
	v := viewmodel.ArticleSummaryViewModel{
		ID:               item.ID,
		Slug:             item.Slug,
		Title:            item.Title,
		Summary:          item.Summary,
		Href:             base + "/" + item.Slug,
		Index:            viewmodel.IndexOf(index),
		DateISO:          iso,
		DateDisplay:      display,
		IsTranslatedByAI: item.IsTranslatedByAI,
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

// groupByMonth groups rows the way the legacy frontend did: newest month first,
// in the order the service already returned them.
func (h *PageHandler) groupByMonth(items []payload.ArticleListItemResponse, lang i18n.Lang, base string) []viewmodel.MonthGroupViewModel {
	groups := make([]viewmodel.MonthGroupViewModel, 0, len(items))
	index := make(map[string]int, len(items))
	for i, item := range items {
		at := item.CreatedAt
		if h.zone != nil {
			at = at.In(h.zone)
		}
		key := at.Format("2006-01")
		groupIndex, ok := index[key]
		if !ok {
			groupIndex = len(groups)
			index[key] = groupIndex
			groups = append(groups, viewmodel.MonthGroupViewModel{
				Label: viewmodel.FormatMonth(at.Year(), at.Month(), lang),
			})
		}
		groups[groupIndex].Items = append(groups[groupIndex].Items, h.summary(item, i, lang, base))
	}
	return groups
}

// filters builds the category chips of the list page.
func (h *PageHandler) filters(ctx context.Context, p listParams) ([]viewmodel.Link, error) {
	categories, err := h.categories.List(ctx, p.Lang.Payload())
	if err != nil {
		return nil, err
	}

	base := viewmodel.SectionPath(p.Lang, "articles")
	query := p.Query()

	out := make([]viewmodel.Link, 0, len(categories)+1)
	out = append(out, viewmodel.Link{
		Label:  h.dict.Message(p.Lang, "articles.all"),
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

// statsView builds the landing page's counters. A missing snapshot is not an
// error: the counter row is simply omitted.
func (h *PageHandler) statsView(ctx context.Context, lang i18n.Lang) []viewmodel.StatViewModel {
	stats, err := h.stats.GetSiteStatistics(ctx)
	if err != nil {
		if !errors.Is(err, service.ErrStatisticsNotFound) {
			log.Printf("read site statistics: %v", err)
		}
		return nil
	}
	return []viewmodel.StatViewModel{
		{Value: strconv.FormatInt(stats.TotalArticles, 10), Label: "stats.articles"},
		{Value: strconv.FormatInt(stats.TotalCategories, 10), Label: "stats.categories"},
		{Value: strconv.FormatInt(stats.TotalTags, 10), Label: "stats.tags"},
		{Value: strconv.FormatInt(stats.TotalArticleViews, 10), Label: "stats.views"},
	}
}

func (h *PageHandler) categoryExists(ctx context.Context, slug string, lang i18n.Lang) bool {
	if slug == "" {
		return true
	}
	_, err := h.categories.GetBySlug(ctx, slug, lang.Payload())
	return err == nil
}

func (h *PageHandler) tagExists(ctx context.Context, slug string) bool {
	if slug == "" {
		return true
	}
	_, err := h.tags.GetBySlug(ctx, slug)
	return err == nil
}

// pageLang is the language of the request path, defaulting to the site default
// when the route somehow did not carry a supported one.
func pageLang(r *http.Request) i18n.Lang {
	lang, ok := i18n.FromPath(r.URL.Path)
	if !ok {
		return i18n.Default()
	}
	return lang
}
