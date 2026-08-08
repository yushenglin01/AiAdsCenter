package repository

import (
	"context"
	"errors"

	"github.com/example/adnova/internal/creative/domain"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("creative not found")

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }
func (r *Repository) Create(ctx context.Context, row *domain.Creative) error {
	return r.db.WithContext(ctx).Create(row).Error
}
func (r *Repository) List(ctx context.Context, tenantID, campaignID string) ([]domain.Creative, error) {
	var rows []domain.Creative
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if campaignID != "" {
		query = query.Where("campaign_id = ?", campaignID)
	}
	err := query.Order("created_at DESC").Find(&rows).Error
	return rows, err
}
func (r *Repository) Get(ctx context.Context, tenantID, id string) (*domain.Creative, error) {
	var row domain.Creative
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) FindByExternalID(ctx context.Context, tenantID, externalID string) (*domain.Creative, error) {
	var row domain.Creative
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND external_id = ?", tenantID, externalID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) Update(ctx context.Context, row *domain.Creative) error {
	return r.db.WithContext(ctx).Save(row).Error
}
func (r *Repository) CampaignExists(ctx context.Context, tenantID, id string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("campaigns").Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, id).Count(&count).Error
	return count == 1, err
}
