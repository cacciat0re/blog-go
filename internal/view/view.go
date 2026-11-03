package view

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"

	"github.com/cacciat0re/blog-go/internal/model/viewmodel"
)

//go:embed templates/*.html
var templatesFS embed.FS

var articleTemplate = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

func RenderArticle(w http.ResponseWriter, viewModel viewmodel.ArticleViewModel) error {
	var buf bytes.Buffer
	if err := articleTemplate.ExecuteTemplate(&buf, "article.html", viewModel); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := buf.WriteTo(w)
	return err
}
