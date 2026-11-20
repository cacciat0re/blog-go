package repository

import (
	"context"

	"gorm.io/gorm"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
)

type ICategoryRepository interface {
	FindAll(ctx context.Context) ([]payload.CategoryResponse, error)
	FindBySlug(ctx context.Context, slug string) (payload.CategoryResponse, error)
}

type CategoryRepository struct {
	db *gorm.DB
}

var _ ICategoryRepository = (*CategoryRepository)(nil)

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) FindAll(ctx context.Context) ([]payload.CategoryResponse, error) {
	items := make([]payload.CategoryResponse, 0)
	err := r.db.WithContext(ctx).
		Table("categories AS c").
		Select("c.id, c.slug, c.name, c.parent_id").
		Order("c.id ASC").
		Scan(&items).Error
	return items, err
}

func (r *CategoryRepository) FindBySlug(ctx context.Context, slug string) (payload.CategoryResponse, error) {
	var item payload.CategoryResponse
	result := r.db.WithContext(ctx).
		Table("categories AS c").
		Select("c.id, c.slug, c.name, c.parent_id").
		Where("c.slug = ?", slug).
		Scan(&item)
	if result.Error != nil {
		return payload.CategoryResponse{}, result.Error
	}
	if result.RowsAffected == 0 {
		return payload.CategoryResponse{}, ErrNotFound
	}
	return item, nil
}
