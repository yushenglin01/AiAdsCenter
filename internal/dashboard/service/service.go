package service

import (
	"context"

	dashboarddomain "github.com/example/adnova/internal/dashboard/domain"
	"github.com/example/adnova/internal/dashboard/repository"
)

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }
func (s *Service) Summary(ctx context.Context, tenantID string) (*dashboarddomain.OperationsSummary, error) {
	return s.repo.Summary(ctx, tenantID)
}
