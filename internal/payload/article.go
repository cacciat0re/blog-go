package payload

import "time"

const (
	DefaultArticleListLimit = 20
	MaxArticleListLimit     = 100
)

// ArticleListRequest is the article list's filter set: an optional keyword, an
// optional category (slug or numeric id) and an optional tag, plus pagination.
type ArticleListRequest struct {
	Key        string `json:"key"`
	Category   string `json:"category"`
	CategoryID *int64 `json:"category_id"`
	Tag        string `json:"tag"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

// ArticleResponse is the published article body. It is the frozen JSON contract
// of the reserved API; the MVC pages use ArticleDetailResponse instead.
type ArticleResponse struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ArticleListItemResponse is a list row: identity, title, summary, status, both
// timestamps, the category and its tags.
type ArticleListItemResponse struct {
	ID        int64             `json:"id"`
	Slug      string            `json:"slug"`
	Title     string            `json:"title"`
	Summary   string            `json:"summary"`
	Status    string            `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Category  *CategoryResponse `json:"category,omitempty"`
	Tags      []TagResponse     `json:"tags"`
}
