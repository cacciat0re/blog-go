package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"gitlab.arorms.cn/arorms/applications/blog/internal/model"
	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
)

var ErrNotFound = errors.New("record not found")

type IArticleRepository interface {
	Create(ctx context.Context, article *model.Article) error
	FindByID(ctx context.Context, id int64) (model.Article, error)
	FindBySlug(ctx context.Context, slug, lang string) (payload.ArticleResponse, error)
	FindPublished(ctx context.Context, query payload.ArticleListRequest) ([]payload.ArticleListItemResponse, int64, error)
	FindPublishedDetail(ctx context.Context, slug, lang string) (payload.ArticleDetailResponse, error)
}

type ArticleRepository struct {
	db *gorm.DB
}

var _ IArticleRepository = (*ArticleRepository)(nil)

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

func (r *ArticleRepository) Create(ctx context.Context, article *model.Article) error {
	return r.db.WithContext(ctx).Omit("Tags", "Category").Create(article).Error
}

func (r *ArticleRepository) FindByID(ctx context.Context, id int64) (model.Article, error) {
	var article model.Article
	err := r.db.WithContext(ctx).
		Preload("Translations").
		Where("id = ?", id).
		First(&article).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Article{}, ErrNotFound
	}
	return article, err
}

func (r *ArticleRepository) FindBySlug(ctx context.Context, slug, lang string) (payload.ArticleResponse, error) {
	var response payload.ArticleResponse
	result := r.db.WithContext(ctx).
		Table("articles").
		Select(`articles.slug,
			at.language AS language,
			at.title,
			at.summary,
			at.html_content AS content,
			at.is_translated_by_ai,
			articles.updated_at`).
		Joins("JOIN article_translations AS at ON at.article_id = articles.id").
		Where("articles.status = ? AND articles.slug = ? AND at.language = ?",
			model.ArticleStatusPublished, slug, lang).
		Scan(&response)
	if result.Error != nil {
		return payload.ArticleResponse{}, result.Error
	}
	if result.RowsAffected == 0 {
		return payload.ArticleResponse{}, ErrNotFound
	}
	return response, nil
}

type articleDetailRow struct {
	ID               int64
	Slug             string
	Language         string
	Title            string
	Summary          string
	Content          string
	Status           string
	IsTranslatedByAI bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CategoryID       *int64
	CategorySlug     *string
	CategoryName     *string
	CategoryParentID *int64
}

// FindPublishedDetail loads one published article with the facets the public
// detail page shows. It shares the list query's join shape so a category name is
// translated the same way it is on a list row.
func (r *ArticleRepository) FindPublishedDetail(ctx context.Context, slug, lang string) (payload.ArticleDetailResponse, error) {
	var row articleDetailRow
	result := r.db.WithContext(ctx).
		Table("articles").
		Select(`articles.id,
			articles.slug,
			articles.status,
			at.language AS language,
			at.title,
			at.summary,
			at.html_content AS content,
			at.is_translated_by_ai,
			articles.created_at,
			articles.updated_at,
			c.id AS category_id,
			c.slug AS category_slug,
			ct.content AS category_name,
			c.parent_id AS category_parent_id`).
		Joins("JOIN article_translations AS at ON at.article_id = articles.id AND at.language = ?", lang).
		Joins("LEFT JOIN categories AS c ON c.id = articles.category_id").
		Joins("LEFT JOIN category_translations AS ct ON ct.category_id = c.id AND ct.language = ?", lang).
		Where("articles.status = ? AND articles.slug = ?", model.ArticleStatusPublished, slug).
		Scan(&row)
	if result.Error != nil {
		return payload.ArticleDetailResponse{}, result.Error
	}
	if result.RowsAffected == 0 {
		return payload.ArticleDetailResponse{}, ErrNotFound
	}

	tagsByArticle, err := r.findTagsByArticleIDs(ctx, []int64{row.ID})
	if err != nil {
		return payload.ArticleDetailResponse{}, err
	}

	response := payload.ArticleDetailResponse{
		ID:               row.ID,
		Slug:             row.Slug,
		Language:         row.Language,
		Title:            row.Title,
		Summary:          row.Summary,
		Content:          row.Content,
		Status:           row.Status,
		IsTranslatedByAI: row.IsTranslatedByAI,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		Tags:             tagsByArticle[row.ID],
	}
	if response.Tags == nil {
		response.Tags = []payload.TagResponse{}
	}
	if row.CategoryID != nil {
		category := &payload.CategoryResponse{ParentID: row.CategoryParentID}
		if row.CategorySlug != nil {
			category.Slug = *row.CategorySlug
		}
		if row.CategoryName != nil {
			category.Name = *row.CategoryName
		}
		response.Category = category
	}
	return response, nil
}

type articleListRow struct {
	ID               int64
	Slug             string
	Title            string
	Summary          string
	Status           string
	IsTranslatedByAI bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CategoryID       *int64
	CategorySlug     *string
	CategoryName     *string
	CategoryParentID *int64
}

// publishedArticlesQuery builds the shared FROM/WHERE conditions. It must not
// carry Select/Order/Limit/Offset because Count reuses it.
func (r *ArticleRepository) publishedArticlesQuery(ctx context.Context, query payload.ArticleListRequest) *gorm.DB {
	db := r.db.WithContext(ctx).
		Model(&model.Article{}).
		Joins("JOIN article_translations AS at ON at.article_id = articles.id AND at.language = ?", query.Language).
		Joins("LEFT JOIN categories AS c ON c.id = articles.category_id").
		Joins("LEFT JOIN category_translations AS ct ON ct.category_id = c.id AND ct.language = ?", query.Language).
		Where("articles.status = ?", model.ArticleStatusPublished)

	if query.Key != "" {
		pattern := "%" + escapeLike(query.Key) + "%"
		db = db.Where(`(at.title ILIKE ? ESCAPE '\' OR at.summary ILIKE ? ESCAPE '\' OR at.markdown_content ILIKE ? ESCAPE '\')`,
			pattern, pattern, pattern)
	}
	if query.Category != "" {
		db = db.Where("c.slug = ?", query.Category)
	}
	if query.CategoryID != nil {
		db = db.Where("articles.category_id = ?", *query.CategoryID)
	}
	if query.Tag != "" {
		db = db.Where(`EXISTS (
			SELECT 1 FROM article_tags AS art
			JOIN tags AS t ON t.id = art.tag_id
			WHERE art.article_id = articles.id AND t.slug = ?)`, query.Tag)
	}
	return db
}

func (r *ArticleRepository) FindPublished(ctx context.Context, query payload.ArticleListRequest) ([]payload.ArticleListItemResponse, int64, error) {
	var total int64
	if err := r.publishedArticlesQuery(ctx, query).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []articleListRow
	err := r.publishedArticlesQuery(ctx, query).
		Select(`articles.id,
			articles.slug,
			articles.status,
			at.title,
			at.summary,
			at.is_translated_by_ai,
			articles.created_at,
			articles.updated_at,
			c.id AS category_id,
			c.slug AS category_slug,
			ct.content AS category_name,
			c.parent_id AS category_parent_id`).
		Order("articles.created_at DESC NULLS LAST").
		Order("articles.id DESC").
		Offset(query.Offset).
		Limit(query.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]payload.ArticleListItemResponse, 0, len(rows))
	if len(rows) == 0 {
		return items, total, nil
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	tagsByArticle, err := r.findTagsByArticleIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	for _, row := range rows {
		item := payload.ArticleListItemResponse{
			ID:               row.ID,
			Slug:             row.Slug,
			Title:            row.Title,
			Summary:          row.Summary,
			Status:           row.Status,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
			IsTranslatedByAI: row.IsTranslatedByAI,
			Tags:             tagsByArticle[row.ID],
		}
		if item.Tags == nil {
			item.Tags = []payload.TagResponse{}
		}
		if row.CategoryID != nil {
			category := &payload.CategoryResponse{ParentID: row.CategoryParentID}
			if row.CategorySlug != nil {
				category.Slug = *row.CategorySlug
			}
			if row.CategoryName != nil {
				category.Name = *row.CategoryName
			}
			item.Category = category
		}
		items = append(items, item)
	}
	return items, total, nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(value string) string {
	return likeEscaper.Replace(value)
}

func (r *ArticleRepository) findTagsByArticleIDs(ctx context.Context, articleIDs []int64) (map[int64][]payload.TagResponse, error) {
	type tagRow struct {
		ArticleID int64
		Slug      string
		Name      string
	}

	var rows []tagRow
	err := r.db.WithContext(ctx).
		Table("article_tags AS art").
		Select("art.article_id, t.slug, t.name").
		Joins("JOIN tags AS t ON t.id = art.tag_id").
		Where("art.article_id IN ?", articleIDs).
		Order("t.name ASC").
		Order("t.slug ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]payload.TagResponse, len(articleIDs))
	for _, row := range rows {
		result[row.ArticleID] = append(result[row.ArticleID], payload.TagResponse{
			Slug: row.Slug,
			Name: row.Name,
		})
	}
	return result, nil
}
