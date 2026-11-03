package viewmodel

import (
	"html/template"
	"time"
)

type ArticleViewModel struct {
	ID             int64
	Title          string
	Summary        string
	Lang           string
	Content        template.HTML
	IsAITranslated bool
	UpdatedAt      time.Time
}

type ArticleListViewModel struct {
}
