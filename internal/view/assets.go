package view

import (
	"io/fs"
	"strings"
)

// assetFS publishes the embedded assets at the addresses the templates use.
//
// On disk the files are grouped by kind, so the Tailwind source can sit next to
// the stylesheet it produces: assets/css/index.css, assets/css/katex/…,
// assets/img/logo.svg. The URL space is flatter and stable:
//
//	/assets/index.css   ← assets/css/index.css
//	/assets/katex/…     ← assets/css/katex/…
//	/assets/img/…       ← assets/img/…
//
// The mapping is a table rather than a directory walk: a path either corresponds
// to a file or it does not, and an unknown path must not accidentally resolve to
// something else that happens to live in the tree.
type assetFS struct {
	files fs.FS // rooted at the package directory, paths look like "assets/css/…"
}

// publish maps a URL path under /assets to embedded paths, most specific first.
// The first path that exists wins; there are no duplicates between the groups.
func (a assetFS) publish(name string) []string {
	switch {
	case name == "index.css":
		return []string{"assets/css/index.css"}
	case strings.HasPrefix(name, "katex/"):
		return []string{"assets/css/" + name}
	case strings.HasPrefix(name, "img/"):
		return []string{"assets/" + name}
	}
	return nil
}

// Open implements fs.FS over URL space. Directories are not published: /assets
// holds files only, so a request for a directory is a 404 rather than a listing.
func (a assetFS) Open(name string) (fs.File, error) {
	for _, candidate := range a.publish(name) {
		if f, err := a.files.Open(candidate); err == nil {
			return f, nil
		}
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
