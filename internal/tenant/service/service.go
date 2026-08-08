package service

import (
	"context"
	"fmt"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/model"
	"github.com/example/adnova/internal/tenant/repository"
	"gorm.io/gorm"
)

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Current(ctx context.Context, tenantID string) (*model.Tenant, error) {
	tenant, err := s.repo.FindByID(ctx, tenantID)
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find tenant: %w", err)
	}
	return tenant, nil
}
