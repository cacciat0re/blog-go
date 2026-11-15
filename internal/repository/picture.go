package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
)

type IPictureRepository interface {
	// FindGallery returns the pictures flagged for the public gallery, newest
	// first, with their tags.
	FindGallery(ctx context.Context, limit, offset int) ([]payload.PictureResponse, int64, error)
}

type PictureRepository struct {
	db *gorm.DB
}

var _ IPictureRepository = (*PictureRepository)(nil)

func NewPictureRepository(db *gorm.DB) *PictureRepository {
	return &PictureRepository{db: db}
}

type pictureRow struct {
	ID               int64
	Filename         string
	OriginalFilename string
	MimeType         string
	Size             int64
	URL              string
	ThumbnailURL     *string
	Alt              *string
	ShowInGallery    bool
	CreatedAt        time.Time
}

func (r *PictureRepository) FindGallery(ctx context.Context, limit, offset int) ([]payload.PictureResponse, int64, error) {
	query := r.db.WithContext(ctx).
		Table("pictures").
		Where("show_in_gallery = TRUE")

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []pictureRow
	if err := query.
		Select("id, filename, original_filename, mime_type, size, url, thumbnail_url, alt, show_in_gallery, created_at").
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]payload.PictureResponse, 0, len(rows))
	for _, row := range rows {
		item := payload.PictureResponse{
			ID:               row.ID,
			Filename:         row.Filename,
			OriginalFilename: row.OriginalFilename,
			MimeType:         row.MimeType,
			Size:             row.Size,
			URL:              row.URL,
			ShowInGallery:    row.ShowInGallery,
			Tags:             []payload.TagResponse{},
			CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339),
		}
		if row.ThumbnailURL != nil {
			item.ThumbnailURL = *row.ThumbnailURL
		}
		if row.Alt != nil {
			item.Alt = *row.Alt
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return items, total, nil
	}

	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	tagsByPicture, err := r.findTagsByPictureIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		if tags := tagsByPicture[items[i].ID]; tags != nil {
			items[i].Tags = tags
		}
	}
	return items, total, nil
}

func (r *PictureRepository) findTagsByPictureIDs(ctx context.Context, pictureIDs []int64) (map[int64][]payload.TagResponse, error) {
	type tagRow struct {
		PictureID int64
		ID        int64
		Slug      string
		Name      string
	}

	var rows []tagRow
	err := r.db.WithContext(ctx).
		Table("picture_tags AS pt").
		Select("pt.picture_id, t.id, t.slug, t.name").
		Joins("JOIN tags AS t ON t.id = pt.tag_id").
		Where("pt.picture_id IN ?", pictureIDs).
		Order("t.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]payload.TagResponse, len(pictureIDs))
	for _, row := range rows {
		result[row.PictureID] = append(result[row.PictureID], payload.TagResponse{
			ID:   row.ID,
			Slug: row.Slug,
			Name: row.Name,
		})
	}
	return result, nil
}
