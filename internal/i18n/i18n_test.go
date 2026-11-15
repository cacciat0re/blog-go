package i18n

import (
	"strings"
	"testing"
)

// TestPacksAreAligned is the guard against a half-translated site: every language
// must define exactly the same keys, so a missing string shows up here instead of
// on a reader's page.
func TestPacksAreAligned(t *testing.T) {
	dict, err := Load()
	if err != nil {
		t.Fatalf("load locales: %v", err)
	}

	base := Supported()[0]
	want := dict.Keys(base)
	if len(want) == 0 {
		t.Fatal("no messages loaded")
	}

	for _, lang := range Supported()[1:] {
		got := dict.Keys(lang)
		if len(got) != len(want) {
			t.Errorf("%s defines %d keys, %s defines %d", lang, len(got), base, len(want))
		}
		set := make(map[string]bool, len(got))
		for _, k := range got {
			set[k] = true
		}
		for _, k := range want {
			if !set[k] {
				t.Errorf("%s is missing key %q", lang, k)
			}
		}
	}
}

func TestMessageFallsBackToKey(t *testing.T) {
	dict, err := Load()
	if err != nil {
		t.Fatalf("load locales: %v", err)
	}
	if got := dict.Message(ZH, "nav.noSuchKey"); got != "nav.noSuchKey" {
		t.Errorf("missing key returned %q, want the key itself", got)
	}
}

func TestTranslatorFormats(t *testing.T) {
	dict, err := Load()
	if err != nil {
		t.Fatalf("load locales: %v", err)
	}
	tr := dict.Translator(EN)
	got := tr("pagination.summary", 2, 7)
	if !strings.Contains(got, "2") || !strings.Contains(got, "7") {
		t.Errorf("formatted message %q lost its arguments", got)
	}
}

func TestLangParsing(t *testing.T) {
	tests := []struct {
		code string
		ok   bool
	}{
		{"en", true},
		{"EN", true},
		{"zh", true},
		{"fr", false},
		{"", false},
		{"zh-CN", false},
	}
	for _, tc := range tests {
		if _, ok := Normalize(tc.code); ok != tc.ok {
			t.Errorf("Normalize(%q) = %t, want %t", tc.code, ok, tc.ok)
		}
	}
}

func TestPaths(t *testing.T) {
	if got := (EN).BasePath(); got != "/en" {
		t.Errorf("BasePath = %q", got)
	}
	if got := ReplaceLang("/en/articles/x", ZH); got != "/zh/articles/x" {
		t.Errorf("ReplaceLang = %q", got)
	}
	if got := StripLang("/en/articles/x", EN); got != "/articles/x" {
		t.Errorf("StripLang = %q", got)
	}
	if got := StripLang("/en", EN); got != "" {
		t.Errorf("StripLang root = %q", got)
	}
	if got := (EN).Other(); got != ZH {
		t.Errorf("Other(EN) = %q", got)
	}
}

func TestFromPath(t *testing.T) {
	if lang, ok := FromPath("/zh/articles/x"); !ok || lang != ZH {
		t.Errorf("FromPath = %q %t", lang, ok)
	}
	if _, ok := FromPath("/api/articles/x"); ok {
		t.Error("FromPath accepted /api")
	}
}
