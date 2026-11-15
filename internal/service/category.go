package service

import (
	"context"
	"errors"
	"fmt"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/repository"
)

var ErrCategoryNotFound = errors.New("category not found")

type ICategoryService interface {
	List(ctx context.Context, lang string) ([]payload.CategoryResponse, error)
	GetBySlug(ctx context.Context, slug, lang string) (payload.CategoryResponse, error)
	// Tree returns the categories nested one level by parent, as the archive page
	// shows them.
	Tree(ctx context.Context, lang string) ([]payload.CategoryTreeNode, error)
}

type CategoryService struct {
	repository repository.ICategoryRepository
}

func NewCategoryService(repo repository.ICategoryRepository) ICategoryService {
	return &CategoryService{repository: repo}
}

func (s *CategoryService) List(ctx context.Context, lang string) ([]payload.CategoryResponse, error) {
	items, err := s.repository.FindAll(ctx, lang)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	for i := range items {
		if items[i].Name == "" {
			items[i].Name = items[i].Slug
		}
	}
	return items, nil
}

func (s *CategoryService) GetBySlug(ctx context.Context, slug, lang string) (payload.CategoryResponse, error) {
	item, err := s.repository.FindBySlug(ctx, slug, lang)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return payload.CategoryResponse{}, ErrCategoryNotFound
		}
		return payload.CategoryResponse{}, fmt.Errorf("find category %q: %w", slug, err)
	}
	if item.Name == "" {
		item.Name = item.Slug
	}
	return item, nil
}

// Tree nests the flat category list one level by parent, which is the shape the
// archive page prints. Categories whose parent is missing stay at the top level
// rather than disappearing.
func (s *CategoryService) Tree(ctx context.Context, lang string) ([]payload.CategoryTreeNode, error) {
	items, err := s.List(ctx, lang)
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]*payload.CategoryTreeNode, len(items))
	for _, item := range items {
		byID[item.ID] = &payload.CategoryTreeNode{ID: item.ID, Name: item.Name, Slug: item.Slug}
	}

	roots := make([]payload.CategoryTreeNode, 0, len(items))
	for _, item := range items {
		node := byID[item.ID]
		if item.ParentID == nil {
			roots = append(roots, *node)
			continue
		}
		parent, ok := byID[*item.ParentID]
		if !ok {
			roots = append(roots, *node)
			continue
		}
		parent.Children = append(parent.Children, *node)
	}
	return roots, nil
}
