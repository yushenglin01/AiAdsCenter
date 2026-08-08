package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/game/domain"
	"github.com/example/adnova/internal/game/dto"
	"github.com/example/adnova/internal/game/repository"
	"github.com/google/uuid"
)

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, tenantID string, req dto.UpsertGameRequest) (*domain.Game, error) {
	if err := validate(req); err != nil {
		return nil, err
	}
	game := &domain.Game{ID: uuid.NewString(), TenantID: tenantID, Code: strings.TrimSpace(req.Code), Name: strings.TrimSpace(req.Name), PackageName: strings.TrimSpace(req.PackageName), Timezone: req.Timezone, Currency: strings.ToUpper(req.Currency), Status: req.Status}
	if err := s.repo.Create(ctx, game); err != nil {
		return nil, fmt.Errorf("create game: %w", err)
	}
	return game, nil
}

func (s *Service) List(ctx context.Context, tenantID string) ([]domain.Game, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *Service) Get(ctx context.Context, tenantID, id string) (*domain.Game, error) {
	game, err := s.repo.Get(ctx, tenantID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound
	}
	return game, err
}

func (s *Service) Update(ctx context.Context, tenantID, id string, req dto.UpsertGameRequest) (*domain.Game, error) {
	if err := validate(req); err != nil {
		return nil, err
	}
	game, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	game.Code, game.Name, game.PackageName = strings.TrimSpace(req.Code), strings.TrimSpace(req.Name), strings.TrimSpace(req.PackageName)
	game.Timezone, game.Currency, game.Status = req.Timezone, strings.ToUpper(req.Currency), req.Status
	if err := s.repo.Update(ctx, game); err != nil {
		return nil, fmt.Errorf("update game: %w", err)
	}
	return game, nil
}

func validate(req dto.UpsertGameRequest) error {
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		return apperror.InvalidArgument
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		return apperror.InvalidArgument
	}
	if len(req.Currency) != 3 {
		return apperror.InvalidArgument
	}
	return nil
}
