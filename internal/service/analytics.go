package service

import (
	"context"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
)

type AnalyticsService struct {
	repo *repository.AnalyticsRepository
}

func NewAnalyticsService(repo *repository.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) Overview(ctx context.Context) (*domain.AnalyticsOverview, error) {
	return s.repo.Overview(ctx)
}

func (s *AnalyticsService) ByService(ctx context.Context) ([]domain.ServiceDistribution, error) {
	return s.repo.ByService(ctx)
}

func (s *AnalyticsService) ByHour(ctx context.Context) ([]domain.HourlyPoint, error) {
	return s.repo.ByHour(ctx)
}

func (s *AnalyticsService) Staff(ctx context.Context) ([]domain.StaffPerf, error) {
	return s.repo.Staff(ctx)
}

func (s *AnalyticsService) Ratings(ctx context.Context, limit int) ([]domain.RatingFeedback, error) {
	return s.repo.Ratings(ctx, limit)
}

func (s *AnalyticsService) Issues(ctx context.Context) ([]domain.IssueDistribution, error) {
	return s.repo.Issues(ctx)
}
