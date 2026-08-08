package service

import (
	"context"

	recommendationdomain "github.com/example/adnova/internal/recommendation/domain"
	"github.com/example/adnova/internal/recommendation/repository"
)

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }
func (s *Service) List(ctx context.Context, tenantID string, filter recommendationdomain.Filter) ([]recommendationdomain.Detail, error) {
	return s.repo.List(ctx, tenantID, filter)
}
func (s *Service) Get(ctx context.Context, tenantID, id string) (*recommendationdomain.Detail, error) {
	return s.repo.Get(ctx, tenantID, id)
}
