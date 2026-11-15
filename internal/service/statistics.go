package service

import (
	"context"
	"errors"
	"fmt"

	"gitlab.arorms.cn/arorms/applications/blog/internal/payload"
	"gitlab.arorms.cn/arorms/applications/blog/internal/repository"
)

// ErrStatisticsNotFound reports that no statistics snapshot exists yet. It is not
// an error for the landing page: the counters are simply omitted.
var ErrStatisticsNotFound = errors.New("statistics not found")

type IStatisticsService interface {
	GetSiteStatistics(ctx context.Context) (payload.SiteStatisticsResponse, error)
}

type StatisticsService struct {
	repository repository.IStatisticsRepository
}

func NewStatisticsService(repo repository.IStatisticsRepository) IStatisticsService {
	return &StatisticsService{repository: repo}
}

func (s *StatisticsService) GetSiteStatistics(ctx context.Context) (payload.SiteStatisticsResponse, error) {
	stats, err := s.repository.LatestSiteStatistics(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return payload.SiteStatisticsResponse{}, ErrStatisticsNotFound
		}
		return payload.SiteStatisticsResponse{}, fmt.Errorf("read site statistics: %w", err)
	}
	return stats, nil
}
