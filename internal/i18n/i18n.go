// Package i18n holds the UI copy for the public site. Every string a reader
// sees that is not article content lives in locales/*.json; templates never
// contain literal wording, so a language can be added without touching HTML.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
)

// Lang is a URL language code. It is also the path prefix of every public page.
type Lang string

const (
	EN Lang = "en"
	ZH Lang = "zh"
)

//go:embed locales/*.json
var localesFS embed.FS

// Default is served from "/" via a permanent redirect.
func Default() Lang { return EN }

// Supported returns the languages in display order.
func Supported() []Lang { return []Lang{EN, ZH} }

func (l Lang) String() string { return string(l) }

// Payload maps a URL code to the uppercase code stored in article_translations
// and used by internal/payload.
func (l Lang) Payload() string { return strings.ToUpper(l.String()) }

// HTMLLang is the BCP 47 tag for <html lang>.
func (l Lang) HTMLLang() string {
	switch l {
	case ZH:
		return "zh-CN"
	default:
		return "en-US"
	}
}

// Label is the language's own name, written in that language. It is not a
// translation: a language switch must be readable by someone who cannot read the
// page they are on.
func (l Lang) Label() string {
	switch l {
	case ZH:
		return "中文"
	default:
		return "English"
	}
}

// Valid reports whether l is a language this site serves.
func (l Lang) Valid() bool {
	for _, s := range Supported() {
		if s == l {
			return true
		}
	}
	return false
}

// Normalize accepts a URL code in any case and returns the canonical form.
func Normalize(code string) (Lang, bool) {
	l := Lang(strings.ToLower(strings.TrimSpace(code)))
	return l, l.Valid()
}

// FromPath returns the language of a "/{lang}/…" path and whether it matched.
func FromPath(path string) (Lang, bool) {
	segs := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segs) == 0 {
		return "", false
	}
	return Normalize(segs[0])
}

// Section returns the first path segment after the language code
// ("articles", "archive", …), or "" for the language root.
func Section(path string) string {
	segs := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segs) < 2 {
		return ""
	}
	return segs[1]
}

// ReplaceLang rewrites the language prefix of path, keeping the rest verbatim.
func ReplaceLang(path string, l Lang) string {
	segs := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segs) > 0 {
		if _, ok := Normalize(segs[0]); ok {
			segs = segs[1:]
		}
	}
	rest := strings.Join(segs, "/")
	if rest == "" {
		return "/" + l.String()
	}
	return "/" + l.String() + "/" + rest
}

// StripLang removes the /{lang} prefix and returns what follows, starting with a
// slash ("/articles/x"), or "" for the language root.
func StripLang(path string, l Lang) string {
	segs := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segs) > 0 {
		if code, ok := Normalize(segs[0]); ok && code == l {
			segs = segs[1:]
		}
	}
	rest := strings.Join(segs, "/")
	if rest == "" {
		return ""
	}
	return "/" + rest
}

// BasePath is the language root, the prefix of every page of that language.
func (l Lang) BasePath() string { return "/" + l.String() }

// Other returns the other supported language, for the language switch link.
func (l Lang) Other() Lang {
	if l == EN {
		return ZH
	}
	return EN
}

// Dictionary is the parsed locale set.
type Dictionary struct {
	messages map[Lang]map[string]string

	mu       sync.Mutex
	reported map[string]bool
}

// Load parses the embedded locale files. It is meant to run once at startup:
// a broken pack is a deployment bug, not something to paper over per request.
func Load() (*Dictionary, error) {
	d := &Dictionary{
		messages: make(map[Lang]map[string]string, len(Supported())),
		reported: make(map[string]bool),
	}
	for _, l := range Supported() {
		name := "locales/" + l.String() + ".json"
		raw, err := localesFS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read locale %s: %w", name, err)
		}
		msgs := make(map[string]string)
		if err := json.Unmarshal(raw, &msgs); err != nil {
			return nil, fmt.Errorf("parse locale %s: %w", name, err)
		}
		if len(msgs) == 0 {
			return nil, fmt.Errorf("locale %s is empty", name)
		}
		d.messages[l] = msgs
	}
	return d, nil
}

// Message returns the translation of key, or key itself when it is missing.
func (d *Dictionary) Message(l Lang, key string) string {
	if msg, ok := d.lookup(l, key); ok {
		return msg
	}
	d.reportMissing(l, key)
	return key
}

// Translator returns the template function bound to l. Missing keys render as
// the key so a gap is visible on the page instead of blanking the text.
func (d *Dictionary) Translator(l Lang) func(key string, args ...any) string {
	return func(key string, args ...any) string {
		msg, ok := d.lookup(l, key)
		if !ok {
			d.reportMissing(l, key)
			return key
		}
		if len(args) == 0 {
			return msg
		}
		return fmt.Sprintf(msg, args...)
	}
}

// Keys returns the sorted key set of l, used by tests to keep packs aligned.
func (d *Dictionary) Keys(l Lang) []string {
	msgs, ok := d.messages[l]
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(msgs))
	for k := range msgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (d *Dictionary) lookup(l Lang, key string) (string, bool) {
	msgs, ok := d.messages[l]
	if !ok {
		return "", false
	}
	msg, ok := msgs[key]
	return msg, ok
}

func (d *Dictionary) reportMissing(l Lang, key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := l.String() + "/" + key
	if d.reported[id] {
		return
	}
	d.reported[id] = true
	log.Printf("i18n: missing message %q for %q", key, l)
}
