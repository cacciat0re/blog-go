package model

// Category is a site category. The name is stored directly on the row.
type Category struct {
	ID       int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Slug     string    `json:"slug" gorm:"column:slug;size:255;not null;uniqueIndex:uq_categories_slug"`
	Name     string    `json:"name" gorm:"column:name;size:255;not null"`
	ParentID *int64    `json:"parent_id" gorm:"column:parent_id;index"`
	Parent   *Category `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
}
