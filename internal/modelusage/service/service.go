package service

import (
	"context"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	modelusagedomain "github.com/example/adnova/internal/modelusage/domain"
	"github.com/example/adnova/internal/modelusage/repository"
)

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }
func (s *Service) List(ctx context.Context, tenantID string, limit int) ([]agentdomain.ModelUsageRecord, error) {
	return s.repo.List(ctx, tenantID, limit)
}
func (s *Service) Summary(ctx context.Context, tenantID string) (*modelusagedomain.Summary, error) {
	return s.repo.Summary(ctx, tenantID)
}
