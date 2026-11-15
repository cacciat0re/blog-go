package repository

import (
	"context"

	"gorm.io/gorm"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
)

type ITagRepository interface {
	FindAll(ctx context.Context) ([]payload.TagResponse, error)
	FindBySlug(ctx context.Context, slug string) (payload.TagResponse, error)
}

type TagRepository struct {
	db *gorm.DB
}

var _ ITagRepository = (*TagRepository)(nil)

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) FindAll(ctx context.Context) ([]payload.TagResponse, error) {
	items := make([]payload.TagResponse, 0)
	err := r.db.WithContext(ctx).
		Table("tags").
		Select("slug, name").
		Order("name ASC").
		Order("slug ASC").
		Scan(&items).Error
	return items, err
}

func (r *TagRepository) FindBySlug(ctx context.Context, slug string) (payload.TagResponse, error) {
	var item payload.TagResponse
	result := r.db.WithContext(ctx).
		Table("tags").
		Select("slug, name").
		Where("slug = ?", slug).
		Scan(&item)
	if result.Error != nil {
		return payload.TagResponse{}, result.Error
	}
	if result.RowsAffected == 0 {
		return payload.TagResponse{}, ErrNotFound
	}
	return item, nil
}
