package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.arorms.cn/arorms/applications/blog/internal/config"
	"gitlab.arorms.cn/arorms/applications/blog/internal/handler"
	"gitlab.arorms.cn/arorms/applications/blog/internal/i18n"
	"gitlab.arorms.cn/arorms/applications/blog/internal/model"
	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/service"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
)

// fakes implements the service interfaces with fixed data, so routing can be
// tested without a database.
type fakeArticles struct{}

func (fakeArticles) Create(ctx context.Context, a *model.Article) error { return nil }
func (fakeArticles) GetByID(ctx context.Context, id int64) (model.Article, error) {
	return model.Article{}, service.ErrArticleNotFound
}
func (fakeArticles) GetBySlug(ctx context.Context, slug, lang string) (payload.ArticleResponse, error) {
	return payload.ArticleResponse{}, service.ErrArticleNotFound
}
func (fakeArticles) GetAllPublished(ctx context.Context, q payload.ArticleListRequest) (payload.PageResponse[payload.ArticleListItemResponse], error) {
	return payload.PageResponse[payload.ArticleListItemResponse]{
		Content: []payload.ArticleListItemResponse{{
			ID:        1,
			Slug:      "hello-world",
			Title:     "Hello World",
			Summary:   "An example article",
			Status:    "PUBLISHED",
			CreatedAt: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
		}},
		Total:      1,
		Limit:      q.Limit,
		Offset:     q.Offset,
		TotalPages: 1,
	}, nil
}
func (fakeArticles) GetPublishedDetail(ctx context.Context, slug, lang string) (payload.ArticleDetailResponse, error) {
	if slug != "hello-world" {
		return payload.ArticleDetailResponse{}, service.ErrArticleNotFound
	}
	return payload.ArticleDetailResponse{
		ID:        1,
		Slug:      slug,
		Language:  lang,
		Title:     "Hello World",
		Summary:   "An example article",
		Content:   "<p>Hello</p>",
		CreatedAt: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
	}, nil
}

type fakeCategories struct{}

func (fakeCategories) List(ctx context.Context, lang string) ([]payload.CategoryResponse, error) {
	return []payload.CategoryResponse{{ID: 1, Slug: "backend", Name: "Backend"}}, nil
}
func (fakeCategories) GetBySlug(ctx context.Context, slug, lang string) (payload.CategoryResponse, error) {
	if slug != "backend" {
		return payload.CategoryResponse{}, service.ErrCategoryNotFound
	}
	return payload.CategoryResponse{ID: 1, Slug: slug, Name: "Backend"}, nil
}
func (fakeCategories) Tree(ctx context.Context, lang string) ([]payload.CategoryTreeNode, error) {
	return []payload.CategoryTreeNode{{ID: 1, Slug: "backend", Name: "Backend"}}, nil
}

type fakeTags struct{}

func (fakeTags) List(ctx context.Context) ([]payload.TagResponse, error) {
	return []payload.TagResponse{{ID: 1, Slug: "go", Name: "Go"}}, nil
}
func (fakeTags) GetBySlug(ctx context.Context, slug string) (payload.TagResponse, error) {
	if slug != "go" {
		return payload.TagResponse{}, service.ErrTagNotFound
	}
	return payload.TagResponse{ID: 1, Slug: slug, Name: "Go"}, nil
}

type fakePictures struct{}

func (fakePictures) GetGallery(ctx context.Context, limit, offset int) (payload.PageResponse[payload.PictureResponse], error) {
	return payload.PageResponse[payload.PictureResponse]{Limit: limit, Offset: offset}, nil
}

type fakeStatistics struct{}

func (fakeStatistics) GetSiteStatistics(ctx context.Context) (payload.SiteStatisticsResponse, error) {
	return payload.SiteStatisticsResponse{TotalArticles: 24}, nil
}

func newRouter(t *testing.T) http.Handler {
	t.Helper()
	dict, err := i18n.Load()
	if err != nil {
		t.Fatalf("load locales: %v", err)
	}
	renderer, err := view.New(dict)
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	pages := handler.NewPageHandler(
		fakeArticles{}, fakeCategories{}, fakeTags{}, fakePictures{}, fakeStatistics{},
		renderer, dict, time.UTC,
	)
	return New(Handlers{
		Article:  handler.NewArticleHandler(fakeArticles{}),
		Page:     pages,
		Renderer: renderer,
	}, config.CORSConfig{})
}

func do(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestPublicPages(t *testing.T) {
	h := newRouter(t)

	tests := []struct {
		method, target string
		status         int
		contentType    string
		contains       string
		location       string
	}{
		{http.MethodGet, "/", http.StatusPermanentRedirect, "text/html", "", "/en"},
		{http.MethodGet, "/EN", http.StatusPermanentRedirect, "text/html", "", "/en"},
		{http.MethodGet, "/en", http.StatusOK, "text/html", "Hello World", ""},
		{http.MethodGet, "/en/", http.StatusOK, "text/html", "Hello World", ""},
		{http.MethodGet, "/zh", http.StatusOK, "text/html", "最新文章", ""},
		{http.MethodGet, "/en/articles", http.StatusOK, "text/html", "Backend", ""},
		{http.MethodGet, "/en/articles?category=backend", http.StatusOK, "text/html", "Hello World", ""},
		{http.MethodGet, "/en/articles?category=nope", http.StatusNotFound, "text/html", "", ""},
		{http.MethodGet, "/en/articles?tag=go", http.StatusOK, "text/html", "Hello World", ""},
		{http.MethodGet, "/en/articles?page=0", http.StatusOK, "text/html", "Hello World", ""},
		{http.MethodGet, "/en/articles?page=5", http.StatusNotFound, "text/html", "", ""},
		{http.MethodGet, "/en/articles/hello-world", http.StatusOK, "text/html", "<p>Hello</p>", ""},
		{http.MethodGet, "/zh/articles/hello-world", http.StatusOK, "text/html", "<p>Hello</p>", ""},
		{http.MethodGet, "/en/articles/missing", http.StatusNotFound, "text/html", "", ""},
		{http.MethodGet, "/en/archive", http.StatusOK, "text/html", "Backend", ""},
		{http.MethodGet, "/en/gallery", http.StatusOK, "text/html", "", ""},
		{http.MethodGet, "/en/about", http.StatusOK, "text/html", "", ""},
		{http.MethodGet, "/fr/", http.StatusNotFound, "text/html", "", ""},
		{http.MethodGet, "/fr/articles", http.StatusNotFound, "text/html", "", ""},
		{http.MethodGet, "/nothing/at/all", http.StatusNotFound, "text/html", "", ""},
		{http.MethodGet, "/assets/index.css", http.StatusOK, "text/css", "", ""},
		{http.MethodGet, "/assets/katex/katex.min.css", http.StatusOK, "text/css", "", ""},
	}

	for _, tc := range tests {
		rec := do(t, h, tc.method, tc.target)
		if rec.Code != tc.status {
			t.Errorf("%s %s = %d, want %d", tc.method, tc.target, rec.Code, tc.status)
			continue
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, tc.contentType) {
			t.Errorf("%s %s content type = %q, want %q", tc.method, tc.target, ct, tc.contentType)
		}
		if tc.contains != "" && !strings.Contains(rec.Body.String(), tc.contains) {
			t.Errorf("%s %s body does not contain %q", tc.method, tc.target, tc.contains)
		}
		if tc.location != "" {
			if got := rec.Header().Get("Location"); got != tc.location {
				t.Errorf("%s %s redirects to %q, want %q", tc.method, tc.target, got, tc.location)
			}
		}
	}
}

// TestReservedAPIIsUntouched pins the behaviour of the four /api routes as they
// were before the frontend work: the same statuses, the same plain-text bodies.
func TestReservedAPIIsUntouched(t *testing.T) {
	h := newRouter(t)

	tests := []struct {
		method, target string
		status         int
	}{
		// The handler reads a {lang} segment this route does not carry, so it
		// answers 400. That is the legacy behaviour and must not change.
		{http.MethodGet, "/api/articles/hello-world", http.StatusBadRequest},
		{http.MethodGet, "/api/admin/articles/1", http.StatusNotFound},
		{http.MethodGet, "/api/nothing", http.StatusNotFound},
		{http.MethodGet, "/api", http.StatusNotFound},
	}

	for _, tc := range tests {
		rec := do(t, h, tc.method, tc.target)
		if rec.Code != tc.status {
			t.Errorf("%s %s = %d, want %d", tc.method, tc.target, rec.Code, tc.status)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
			t.Errorf("%s %s content type = %q, want plain text", tc.method, tc.target, ct)
		}
	}
}

// TestAPIGroupIsSeparate pins the structural split: /api is its own route group
// with its own plain-text 404, so nothing under it can ever reach a page handler
// or a page renderer.
func TestAPIGroupIsSeparate(t *testing.T) {
	h := newRouter(t)

	// A POST route that exists in the group must still reach its handler.
	if rec := do(t, h, http.MethodPost, "/api/articles"); rec.Code != http.StatusBadRequest {
		t.Errorf("POST /api/articles = %d, want the handler's 400", rec.Code)
	}

	// Every unmatched path under /api answers plain text, whatever its shape.
	for _, target := range []string{"/api", "/api/", "/api/nothing", "/api/articles/a/b/c"} {
		rec := do(t, h, http.MethodGet, target)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
			t.Errorf("GET %s content type = %q, want plain text", target, ct)
		}
		if strings.Contains(rec.Body.String(), "<!DOCTYPE") {
			t.Errorf("GET %s rendered a page", target)
		}
	}
}

// TestUnknownLanguageIsNotFound makes sure a language code that is not supported
// cannot render an English page at a made-up address, which would be a soft 404
// for a crawler.
func TestUnknownLanguageIsNotFound(t *testing.T) {
	h := newRouter(t)

	for _, target := range []string{"/fr", "/fr/articles", "/fr/articles/hello-world", "/de/archive"} {
		rec := do(t, h, http.MethodGet, target)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "noindex") {
			t.Errorf("GET %s 404 page is indexable", target)
		}
	}
}

func TestLanguageIsolation(t *testing.T) {
	h := newRouter(t)

	en := do(t, h, http.MethodGet, "/en")
	zh := do(t, h, http.MethodGet, "/zh")

	if strings.Contains(en.Body.String(), "最新文章") {
		t.Error("english page contains chinese interface copy")
	}
	if strings.Contains(zh.Body.String(), "Latest articles") {
		t.Error("chinese page contains english interface copy")
	}
	if !strings.Contains(en.Body.String(), `lang="en-US"`) || !strings.Contains(zh.Body.String(), `lang="zh-CN"`) {
		t.Error("html lang attribute does not follow the path")
	}
}

func TestIndexable(t *testing.T) {
	h := newRouter(t)

	// The fake gallery is empty, so it must stay out of the index.
	if body := do(t, h, http.MethodGet, "/en/gallery").Body.String(); !bodyMay(body, "noindex") {
		t.Error("empty gallery is indexable")
	}
	if body := do(t, h, http.MethodGet, "/en/articles?key=x").Body.String(); strings.Contains(body, "noindex") {
		t.Error("filtered list is noindex; it should be a normal page")
	}
	if body := do(t, h, http.MethodGet, "/en/articles/missing").Body.String(); !bodyMay(body, "noindex") {
		t.Error("404 page is indexable")
	}
}

func bodyMay(body, needle string) bool { return strings.Contains(body, needle) }
