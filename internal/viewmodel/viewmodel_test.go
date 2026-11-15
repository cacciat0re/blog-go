package viewmodel

import (
	"strings"
	"testing"
	"time"

	"gitlab.arorms.cn/arorms/applications/blog/internal/i18n"
)

func TestListQueryURL(t *testing.T) {
	q := ListQuery{Lang: "en", Page: 1, Limit: DefaultListLimit}

	if got := q.URL("/en/articles", 1); got != "/en/articles" {
		t.Errorf("first page with defaults = %q", got)
	}
	if got := q.URL("/en/articles", 2); got != "/en/articles?page=2" {
		t.Errorf("second page = %q", got)
	}

	filtered := ListQuery{Lang: "en", Key: "go", Category: "backend", Page: 1, Limit: DefaultListLimit}
	got := filtered.URL("/en/articles", 2)
	for _, want := range []string{"category=backend", "key=go", "page=2"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q does not preserve %q", got, want)
		}
	}

	// A non-default limit has to survive as well, or pagination would silently
	// fall back to twenty rows per page.
	custom := ListQuery{Lang: "en", Page: 1, Limit: 5}
	if got := custom.URL("/en/articles", 1); got != "/en/articles?limit=5" {
		t.Errorf("custom limit = %q", got)
	}
}

func TestListQueryDimensionHelpers(t *testing.T) {
	q := ListQuery{Lang: "en", Key: "k", Category: "cat", Tag: "tag", Page: 3, Limit: 20}

	if got := q.WithCategory("other"); got.Page != 1 || got.Category != "other" || got.Tag != "tag" {
		t.Errorf("WithCategory = %+v", got)
	}
	if got := q.Without("category"); got.Page != 1 || got.Category != "" || got.Tag != "tag" || got.Key != "k" {
		t.Errorf("Without(category) = %+v", got)
	}
}

func TestPaginationWindow(t *testing.T) {
	q := ListQuery{Lang: "en", Page: 1, Limit: 20}

	p := NewPagination("/en/articles", q, 1, 1, 3)
	if p == nil {
		t.Fatal("pagination for one page was nil")
	}
	if p.Prev != nil || p.Next != nil {
		t.Errorf("single page has movement links: %+v", p)
	}

	mid := NewPagination("/en/articles", q, 5, 10, 200)
	if mid.Prev == nil || mid.Next == nil || mid.First == nil || mid.Last == nil {
		t.Errorf("mid page is missing links: %+v", mid)
	}
	if len(mid.Pages) != 5 {
		t.Errorf("window = %d pages, want 5", len(mid.Pages))
	}
	if mid.Pages[2].Label != "5" || !mid.Pages[2].Active {
		t.Errorf("window does not centre on page 5: %+v", mid.Pages)
	}

	if NewPagination("/en/articles", q, 1, 0, 0) != nil {
		t.Error("pagination without pages should be nil")
	}
}

func TestNavActiveSection(t *testing.T) {
	nav := NewNav(i18n.EN, "/en/articles/hello-world")
	active := ""
	for _, s := range nav.Sections {
		if s.Active {
			active = s.LabelKey
		}
	}
	if active != "nav.articles" {
		t.Errorf("active section = %q", active)
	}

	home := NewNav(i18n.ZH, "/zh")
	for _, s := range home.Sections {
		if s.Active && s.LabelKey != "nav.home" {
			t.Errorf("language root marked %q active", s.LabelKey)
		}
	}

	if nav.SwitchLang.Href != "/zh/articles/hello-world" {
		t.Errorf("switch link = %q", nav.SwitchLang.Href)
	}
}

func TestFormatDate(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	at := time.Date(2026, 11, 3, 7, 3, 33, 0, time.UTC)

	iso, display := FormatDate(at, i18n.EN, zone)
	if iso != "2026-11-03T07:03:33Z" {
		t.Errorf("iso = %q", iso)
	}
	if display != "November 3, 2026" {
		t.Errorf("display = %q", display)
	}

	if _, zh := FormatDate(at, i18n.ZH, zone); zh != "2026年11月03日" {
		t.Errorf("zh display = %q", zh)
	}

	if iso, display := FormatDate(time.Time{}, i18n.EN, zone); iso != "" || display != "" {
		t.Errorf("zero time rendered as %q %q", iso, display)
	}
}

func TestTruncateIsRuneSafe(t *testing.T) {
	// Truncating by bytes would split a Chinese character in half.
	got := Truncate("你好，世界这是一个很长的摘要", 5)
	if got != "你好，世界…" {
		t.Errorf("truncate = %q", got)
	}
	if got := Truncate("short", 10); got != "short" {
		t.Errorf("truncate under limit = %q", got)
	}
}

func TestIndexOfPads(t *testing.T) {
	if got := IndexOf(0); got != "01" {
		t.Errorf("IndexOf(0) = %q", got)
	}
	if got := IndexOf(11); got != "12" {
		t.Errorf("IndexOf(11) = %q", got)
	}
}
