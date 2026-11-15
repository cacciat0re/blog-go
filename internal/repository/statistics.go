package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
)

type IStatisticsRepository interface {
	// LatestSiteStatistics returns the most recent snapshot, or ErrNotFound when
	// no snapshot has ever been taken.
	LatestSiteStatistics(ctx context.Context) (payload.SiteStatisticsResponse, error)
}

type StatisticsRepository struct {
	db *gorm.DB
}

var _ IStatisticsRepository = (*StatisticsRepository)(nil)

func NewStatisticsRepository(db *gorm.DB) *StatisticsRepository {
	return &StatisticsRepository{db: db}
}

func (r *StatisticsRepository) LatestSiteStatistics(ctx context.Context) (payload.SiteStatisticsResponse, error) {
	var row struct {
		TotalArticles     int64
		TotalCategories   int64
		TotalTags         int64
		TotalArticleViews int64
		Date              time.Time
	}

	result := r.db.WithContext(ctx).
		Table("site_statistics").
		Select("total_articles, total_categories, total_tags, total_article_views, date").
		Order("date DESC").
		Order("id DESC").
		Limit(1).
		Scan(&row)
	if result.Error != nil {
		return payload.SiteStatisticsResponse{}, result.Error
	}
	if result.RowsAffected == 0 {
		return payload.SiteStatisticsResponse{}, ErrNotFound
	}

	return payload.SiteStatisticsResponse{
		TotalArticles:     row.TotalArticles,
		TotalCategories:   row.TotalCategories,
		TotalTags:         row.TotalTags,
		TotalArticleViews: row.TotalArticleViews,
		Date:              row.Date.UTC().Format(time.RFC3339),
	}, nil
}
