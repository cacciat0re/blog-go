package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cacciat0re/blog-go/internal/model"
	"github.com/cacciat0re/blog-go/internal/model/viewmodel"
	"github.com/cacciat0re/blog-go/internal/repository"
)

var ErrArticleNotFound = errors.New("article not found")

type IArticleService interface {
	Create(ctx context.Context, article *model.Article) error
	GetByID(ctx context.Context, id int64) (model.Article, error)
	GetBySlug(ctx context.Context, slug, lang string) (viewmodel.ArticleViewModel, error)
}

type ArticleService struct {
	repository repository.IArticleRepository
}

func NewArticleService(repo repository.IArticleRepository) IArticleService {
	return &ArticleService{repository: repo}
}

func (s *ArticleService) Create(ctx context.Context, article *model.Article) error {
	article.Status = strings.ToUpper(strings.TrimSpace(article.Status))
	if article.Status == "" {
		article.Status = model.ArticleStatusDraft
	}
	for i := range article.Translations {
		article.Translations[i].Language = strings.ToUpper(strings.TrimSpace(article.Translations[i].Language))
	}

	if err := s.repository.Create(ctx, article); err != nil {
		return fmt.Errorf("create article: %w", err)
	}
	return nil
}

func (s *ArticleService) GetByID(ctx context.Context, id int64) (model.Article, error) {
	article, err := s.repository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Article{}, ErrArticleNotFound
		}
		return model.Article{}, fmt.Errorf("find article %d: %w", id, err)
	}
	return article, nil
}

func (s *ArticleService) GetBySlug(ctx context.Context, slug, lang string) (viewmodel.ArticleViewModel, error) {
	lang = strings.ToUpper(strings.TrimSpace(lang))
	if lang == "" {
		lang = "EN"
	}

	resonse, err := s.repository.FindBySlug(ctx, slug, lang)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return viewmodel.ArticleViewModel{}, ErrArticleNotFound
		}
		return viewmodel.ArticleViewModel{}, fmt.Errorf("find published article %q: %w", slug, err)
	}

	return resonse, nil
}
