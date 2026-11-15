package service

import (
	"context"
	"fmt"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/repository"
)

type IPictureService interface {
	// GetGallery returns one page of gallery pictures plus the total count.
	GetGallery(ctx context.Context, limit, offset int) (payload.PageResponse[payload.PictureResponse], error)
}

type PictureService struct {
	repository repository.IPictureRepository
}

func NewPictureService(repo repository.IPictureRepository) IPictureService {
	return &PictureService{repository: repo}
}

func (s *PictureService) GetGallery(ctx context.Context, limit, offset int) (payload.PageResponse[payload.PictureResponse], error) {
	items, total, err := s.repository.FindGallery(ctx, limit, offset)
	if err != nil {
		return payload.PageResponse[payload.PictureResponse]{}, fmt.Errorf("list gallery pictures: %w", err)
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	return payload.PageResponse[payload.PictureResponse]{
		Content:    items,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		TotalPages: totalPages,
		HasNext:    int64(offset+len(items)) < total,
	}, nil
}
