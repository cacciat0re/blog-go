package payload

type CategoryResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ParentID *int64 `json:"parent_id"`
}

// CategoryTreeNode is a category with its children, for the archive index.
type CategoryTreeNode struct {
	ID       int64              `json:"id"`
	Name     string             `json:"name"`
	Slug     string             `json:"slug"`
	Children []CategoryTreeNode `json:"children"`
}
