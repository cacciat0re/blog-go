package payload

import "time"

// ArticleDetailResponse is the published article as the public detail page needs
// it: the article's own timestamps and status, the body, and the category and
// tags it belongs to.
//
// It is separate from ArticleResponse on purpose. ArticleResponse is the frozen
// JSON contract of the reserved API; this type serves the MVC page only and may
// grow without changing what /api returns.
type ArticleDetailResponse struct {
	ID        int64             `json:"id"`
	Slug      string            `json:"slug"`
	Title     string            `json:"title"`
	Summary   string            `json:"summary"`
	Content   string            `json:"content"`
	Status    string            `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Category  *CategoryResponse `json:"category,omitempty"`
	Tags      []TagResponse     `json:"tags"`
}
