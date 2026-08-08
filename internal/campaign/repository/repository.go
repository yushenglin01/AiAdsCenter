package repository

import (
	"context"
	"errors"

	"github.com/example/adnova/internal/campaign/domain"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("campaign resource not found")

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateChannel(ctx context.Context, channel *domain.Channel) error {
	return r.db.WithContext(ctx).Create(channel).Error
}
func (r *Repository) ListChannels(ctx context.Context, tenantID string) ([]domain.Channel, error) {
	var rows []domain.Channel
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("name").Find(&rows).Error
	return rows, err
}
func (r *Repository) GetChannel(ctx context.Context, tenantID, id string) (*domain.Channel, error) {
	var row domain.Channel
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) UpdateChannel(ctx context.Context, channel *domain.Channel) error {
	return r.db.WithContext(ctx).Save(channel).Error
}

func (r *Repository) CreateCampaign(ctx context.Context, campaign *domain.Campaign) error {
	return r.db.WithContext(ctx).Create(campaign).Error
}
func (r *Repository) ListCampaigns(ctx context.Context, tenantID, gameID string) ([]domain.Campaign, error) {
	var rows []domain.Campaign
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if gameID != "" {
		query = query.Where("game_id = ?", gameID)
	}
	err := query.Order("created_at DESC").Find(&rows).Error
	return rows, err
}
func (r *Repository) GetCampaign(ctx context.Context, tenantID, id string) (*domain.Campaign, error) {
	var row domain.Campaign
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) FindCampaignByExternalID(ctx context.Context, tenantID, externalID string) (*domain.Campaign, error) {
	var row domain.Campaign
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND external_id = ?", tenantID, externalID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) UpdateCampaign(ctx context.Context, campaign *domain.Campaign) error {
	return r.db.WithContext(ctx).Save(campaign).Error
}

func (r *Repository) GameExists(ctx context.Context, tenantID, id string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("games").Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, id).Count(&count).Error
	return count == 1, err
}
