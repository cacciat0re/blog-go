package service

import (
	"context"
	"errors"
	"fmt"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/repository"
)

var ErrTagNotFound = errors.New("tag not found")

type ITagService interface {
	List(ctx context.Context) ([]payload.TagResponse, error)
	GetBySlug(ctx context.Context, slug string) (payload.TagResponse, error)
}

type TagService struct {
	repository repository.ITagRepository
}

func NewTagService(repo repository.ITagRepository) ITagService {
	return &TagService{repository: repo}
}

func (s *TagService) List(ctx context.Context) ([]payload.TagResponse, error) {
	items, err := s.repository.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return items, nil
}

func (s *TagService) GetBySlug(ctx context.Context, slug string) (payload.TagResponse, error) {
	item, err := s.repository.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return payload.TagResponse{}, ErrTagNotFound
		}
		return payload.TagResponse{}, fmt.Errorf("find tag %q: %w", slug, err)
	}
	return item, nil
}
