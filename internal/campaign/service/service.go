package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/example/adnova/internal/campaign/domain"
	"github.com/example/adnova/internal/campaign/dto"
	"github.com/example/adnova/internal/campaign/repository"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) CreateChannel(ctx context.Context, tenantID string, req dto.UpsertChannelRequest) (*domain.Channel, error) {
	row := &domain.Channel{ID: uuid.NewString(), TenantID: tenantID, Code: strings.ToUpper(strings.TrimSpace(req.Code)), Name: strings.TrimSpace(req.Name), Provider: strings.ToUpper(strings.TrimSpace(req.Provider)), Status: req.Status}
	if row.Code == "" || row.Name == "" {
		return nil, apperror.InvalidArgument
	}
	if err := s.repo.CreateChannel(ctx, row); err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}
	return row, nil
}
func (s *Service) ListChannels(ctx context.Context, tenantID string) ([]domain.Channel, error) {
	return s.repo.ListChannels(ctx, tenantID)
}
func (s *Service) GetChannel(ctx context.Context, tenantID, id string) (*domain.Channel, error) {
	row, err := s.repo.GetChannel(ctx, tenantID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound
	}
	return row, err
}
func (s *Service) UpdateChannel(ctx context.Context, tenantID, id string, req dto.UpsertChannelRequest) (*domain.Channel, error) {
	row, err := s.GetChannel(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	row.Code, row.Name, row.Provider, row.Status = strings.ToUpper(strings.TrimSpace(req.Code)), strings.TrimSpace(req.Name), strings.ToUpper(strings.TrimSpace(req.Provider)), req.Status
	if row.Code == "" || row.Name == "" {
		return nil, apperror.InvalidArgument
	}
	if err := s.repo.UpdateChannel(ctx, row); err != nil {
		return nil, fmt.Errorf("update channel: %w", err)
	}
	return row, nil
}

func (s *Service) CreateCampaign(ctx context.Context, tenantID string, req dto.UpsertCampaignRequest) (*domain.Campaign, error) {
	budget, err := validateCampaign(req)
	if err != nil {
		return nil, err
	}
	if err := s.validateReferences(ctx, tenantID, req.GameID, req.ChannelID); err != nil {
		return nil, err
	}
	row := &domain.Campaign{ID: uuid.NewString(), TenantID: tenantID, GameID: req.GameID, ChannelID: req.ChannelID, ExternalID: strings.TrimSpace(req.ExternalID), Name: strings.TrimSpace(req.Name), Country: strings.ToUpper(req.Country), DailyBudget: budget, Currency: strings.ToUpper(req.Currency), Status: req.Status}
	if err := s.repo.CreateCampaign(ctx, row); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}
	return row, nil
}
func (s *Service) ListCampaigns(ctx context.Context, tenantID, gameID string) ([]domain.Campaign, error) {
	return s.repo.ListCampaigns(ctx, tenantID, gameID)
}
func (s *Service) GetCampaign(ctx context.Context, tenantID, id string) (*domain.Campaign, error) {
	row, err := s.repo.GetCampaign(ctx, tenantID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound
	}
	return row, err
}
func (s *Service) UpdateCampaign(ctx context.Context, tenantID, id string, req dto.UpsertCampaignRequest) (*domain.Campaign, error) {
	budget, err := validateCampaign(req)
	if err != nil {
		return nil, err
	}
	if err := s.validateReferences(ctx, tenantID, req.GameID, req.ChannelID); err != nil {
		return nil, err
	}
	row, err := s.GetCampaign(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	row.GameID, row.ChannelID, row.ExternalID, row.Name = req.GameID, req.ChannelID, strings.TrimSpace(req.ExternalID), strings.TrimSpace(req.Name)
	row.Country, row.DailyBudget, row.Currency, row.Status = strings.ToUpper(req.Country), budget, strings.ToUpper(req.Currency), req.Status
	if err := s.repo.UpdateCampaign(ctx, row); err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}
	return row, nil
}

func (s *Service) validateReferences(ctx context.Context, tenantID, gameID, channelID string) error {
	gameOK, err := s.repo.GameExists(ctx, tenantID, gameID)
	if err != nil {
		return err
	}
	if !gameOK {
		return apperror.InvalidArgument
	}
	if _, err := s.repo.GetChannel(ctx, tenantID, channelID); err != nil {
		return apperror.InvalidArgument
	}
	return nil
}
func validateCampaign(req dto.UpsertCampaignRequest) (decimal.Decimal, error) {
	budget, err := decimal.NewFromString(req.DailyBudget)
	if err != nil || budget.IsNegative() || strings.TrimSpace(req.ExternalID) == "" || strings.TrimSpace(req.Name) == "" || len(req.Country) != 2 || len(req.Currency) != 3 {
		return decimal.Zero, apperror.InvalidArgument
	}
	return budget, nil
}
