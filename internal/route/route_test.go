package route

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.arorms.cn/arorms/applications/blog/internal/config"
	"gitlab.arorms.cn/arorms/applications/blog/internal/handler"
	"gitlab.arorms.cn/arorms/applications/blog/internal/handler/api"
	"gitlab.arorms.cn/arorms/applications/blog/internal/model"
	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/service"
	"gitlab.arorms.cn/arorms/applications/blog/internal/view"
)

type fakeArticles struct{}

func (fakeArticles) Create(context.Context, *model.Article) error { return nil }
func (fakeArticles) GetByID(context.Context, int64) (model.Article, error) {
	return model.Article{}, service.ErrArticleNotFound
}
func (fakeArticles) GetBySlug(context.Context, string) (payload.ArticleResponse, error) {
	return payload.ArticleResponse{}, service.ErrArticleNotFound
}
func (fakeArticles) GetAllPublished(_ context.Context, q payload.ArticleListRequest) (payload.PageResponse[payload.ArticleListItemResponse], error) {
	return payload.PageResponse[payload.ArticleListItemResponse]{
		Content: []payload.ArticleListItemResponse{{
			ID:        1,
			Slug:      "hello-world",
			Title:     "你好，世界",
			Summary:   "一篇示例文章",
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
func (fakeArticles) GetPublishedDetail(_ context.Context, slug string) (payload.ArticleDetailResponse, error) {
	if slug != "hello-world" {
		return payload.ArticleDetailResponse{}, service.ErrArticleNotFound
	}
	return payload.ArticleDetailResponse{
		ID:        1,
		Slug:      slug,
		Title:     "你好，世界",
		Summary:   "一篇示例文章",
		Content:   "<p>你好</p>",
		CreatedAt: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
	}, nil
}

type fakeCategories struct{}

func (fakeCategories) List(context.Context) ([]payload.CategoryResponse, error) {
	return []payload.CategoryResponse{{ID: 1, Slug: "backend", Name: "后端"}}, nil
}
func (fakeCategories) GetBySlug(_ context.Context, slug string) (payload.CategoryResponse, error) {
	if slug != "backend" {
		return payload.CategoryResponse{}, service.ErrCategoryNotFound
	}
	return payload.CategoryResponse{ID: 1, Slug: slug, Name: "后端"}, nil
}
func (fakeCategories) Tree(context.Context) ([]payload.CategoryTreeNode, error) {
	return []payload.CategoryTreeNode{{ID: 1, Slug: "backend", Name: "后端"}}, nil
}

type fakeTags struct{}

func (fakeTags) List(context.Context) ([]payload.TagResponse, error) {
	return []payload.TagResponse{{ID: 1, Slug: "go", Name: "Go"}}, nil
}
func (fakeTags) GetBySlug(_ context.Context, slug string) (payload.TagResponse, error) {
	if slug != "go" {
		return payload.TagResponse{}, service.ErrTagNotFound
	}
	return payload.TagResponse{ID: 1, Slug: slug, Name: "Go"}, nil
}

type fakePictures struct{}

func (fakePictures) GetGallery(_ context.Context, limit, offset int) (payload.PageResponse[payload.PictureResponse], error) {
	return payload.PageResponse[payload.PictureResponse]{
		Limit:  limit,
		Offset: offset,
	}, nil
}

func newRouter(t *testing.T) http.Handler {
	t.Helper()
	renderer, err := view.New()
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	handler.Init(renderer, time.UTC)

	articles := fakeArticles{}
	categories := fakeCategories{}
	tags := fakeTags{}
	pictures := fakePictures{}

	return New(Handlers{
		MVC: MVC{
			Home:     handler.NewHomeHandler(articles),
			Articles: handler.NewArticlesHandler(articles, categories, tags),
			Gallery:  handler.NewGalleryHandler(pictures),
			About:    handler.NewAboutHandler(),
		},
		API: API{Article: api.NewArticleAPIHandler(articles)},
	}, config.CORSConfig{})
}

func do(t *testing.T, h http.Handler, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRootPages(t *testing.T) {
	h := newRouter(t)
	for _, target := range []string{"/", "/articles", "/articles/hello-world", "/gallery", "/about"} {
		rec := do(t, h, target, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", target, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("GET %s content type = %q", target, ct)
		}
	}
}

func TestPagesRenderChinese(t *testing.T) {
	h := newRouter(t)

	body := do(t, h, "/", nil).Body.String()
	for _, want := range []string{`<html lang="zh-CN">`, "最新文章", "你好，世界"} {
		if !strings.Contains(body, want) {
			t.Errorf("home body missing %q", want)
		}
	}

	body = do(t, h, "/articles", nil).Body.String()
	if !strings.Contains(body, "你好，世界") {
		t.Error("article list did not render the article")
	}
}

func TestUnknownPathsAreNotFound(t *testing.T) {
	h := newRouter(t)
	for _, target := range []string{"/en", "/en/", "/en/articles", "/zh", "/zh/articles", "/archive", "/missing"} {
		rec := do(t, h, target, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "" {
			t.Errorf("GET %s redirected to %q, want no redirect", target, loc)
		}
		if !strings.Contains(rec.Body.String(), "页面未找到") {
			t.Errorf("GET %s did not render the error page", target)
		}
	}
}

func TestLinksAreUnprefixed(t *testing.T) {
	h := newRouter(t)
	body := do(t, h, "/", nil).Body.String()

	for _, want := range []string{`href="/"`, `href="/articles"`, `href="/gallery"`, `href="/about"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
	for _, unwanted := range []string{`href="/en/`, `href="/zh/`} {
		if strings.Contains(body, unwanted) {
			t.Errorf("body still contains %q", unwanted)
		}
	}
}

func TestAssetsAndAPI(t *testing.T) {
	h := newRouter(t)

	if rec := do(t, h, "/assets/index.css", nil); rec.Code != http.StatusOK {
		t.Errorf("GET index.css = %d", rec.Code)
	}

	rec := do(t, h, "/api/nothing", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /api/nothing = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("/api 404 content type = %q", ct)
	}
}

func TestRobotsTXT(t *testing.T) {
	h := newRouter(t)

	rec := do(t, h, "/robots.txt", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /robots.txt = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"User-agent: *", "Allow: /"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestResponseHeaders(t *testing.T) {
	h := newRouter(t)

	rec := do(t, h, "/", nil)
	if got := rec.Header().Get("Content-Language"); got != "zh-CN" {
		t.Errorf("Content-Language = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-cache") {
		t.Errorf("Cache-Control = %q", got)
	}

	rec = do(t, h, "/missing", nil)
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("404 Cache-Control = %q", got)
	}
}
