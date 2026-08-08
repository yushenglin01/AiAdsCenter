package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/creative/domain"
	"github.com/example/adnova/internal/creative/dto"
	"github.com/example/adnova/internal/creative/repository"
	"github.com/google/uuid"
)

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ctx context.Context, tenantID string, req dto.UpsertCreativeRequest) (*domain.Creative, error) {
	if strings.TrimSpace(req.ExternalID) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, apperror.InvalidArgument
	}
	ok, err := s.repo.CampaignExists(ctx, tenantID, req.CampaignID)
	if err != nil || !ok {
		return nil, apperror.InvalidArgument
	}
	row := &domain.Creative{ID: uuid.NewString(), TenantID: tenantID, CampaignID: req.CampaignID, ExternalID: strings.TrimSpace(req.ExternalID), Name: strings.TrimSpace(req.Name), Type: req.Type, Status: req.Status}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, fmt.Errorf("create creative: %w", err)
	}
	return row, nil
}
func (s *Service) List(ctx context.Context, tenantID, campaignID string) ([]domain.Creative, error) {
	return s.repo.List(ctx, tenantID, campaignID)
}
func (s *Service) Get(ctx context.Context, tenantID, id string) (*domain.Creative, error) {
	row, err := s.repo.Get(ctx, tenantID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound
	}
	return row, err
}
func (s *Service) Update(ctx context.Context, tenantID, id string, req dto.UpsertCreativeRequest) (*domain.Creative, error) {
	ok, err := s.repo.CampaignExists(ctx, tenantID, req.CampaignID)
	if err != nil || !ok {
		return nil, apperror.InvalidArgument
	}
	row, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	row.CampaignID, row.ExternalID, row.Name, row.Type, row.Status = req.CampaignID, strings.TrimSpace(req.ExternalID), strings.TrimSpace(req.Name), req.Type, req.Status
	if row.ExternalID == "" || row.Name == "" {
		return nil, apperror.InvalidArgument
	}
	if err := s.repo.Update(ctx, row); err != nil {
		return nil, fmt.Errorf("update creative: %w", err)
	}
	return row, nil
}
