package repository

import (
	"context"
	"errors"
	"time"

	"github.com/example/adnova/internal/common/apperror"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ScopeExists(ctx context.Context, tenantID, gameID, campaignID string) (bool, error) {
	query := r.db.WithContext(ctx).Table("games").Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, gameID)
	if campaignID != "" {
		query = r.db.WithContext(ctx).Table("campaigns").Where("tenant_id = ? AND game_id = ? AND id = ? AND deleted_at IS NULL", tenantID, gameID, campaignID)
	}
	var count int64
	err := query.Count(&count).Error
	return count == 1, err
}

func (r *Repository) FindByHash(ctx context.Context, tenantID, hash string) (*researchdomain.Source, error) {
	var row researchdomain.Source
	result := r.db.WithContext(ctx).Where("tenant_id = ? AND content_hash = ?", tenantID, hash).Limit(1).Find(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *Repository) Create(ctx context.Context, row *researchdomain.Source) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) Get(ctx context.Context, tenantID, id string) (*researchdomain.Source, error) {
	var row researchdomain.Source
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.NotFound
	}
	return &row, err
}

func (r *Repository) List(ctx context.Context, tenantID string, filter researchdomain.Filter) ([]researchdomain.Source, error) {
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if filter.GameID != "" {
		query = query.Where("game_id = ?", filter.GameID)
	}
	if filter.CampaignID != "" {
		query = query.Where("campaign_id = ?", filter.CampaignID)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []researchdomain.Source
	err := query.Order("published_at DESC, created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) Decide(ctx context.Context, tenantID, id, status, comment, actorID string) (*researchdomain.Source, error) {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&researchdomain.Source{}).
		Where("tenant_id = ? AND id = ? AND status = ?", tenantID, id, researchdomain.StatusPending).
		Updates(map[string]any{"status": status, "review_comment": comment, "reviewed_by": actorID, "reviewed_at": now})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		row, err := r.Get(ctx, tenantID, id)
		if err != nil {
			return nil, err
		}
		if row.Status != status {
			return nil, apperror.Conflict
		}
		return row, nil
	}
	return r.Get(ctx, tenantID, id)
}

func (r *Repository) SearchVerified(ctx context.Context, tenantID, gameID, campaignID string, through time.Time, limit int) ([]researchdomain.Source, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	query := r.db.WithContext(ctx).Where("tenant_id = ? AND status = ? AND published_at <= ?", tenantID, researchdomain.StatusVerified, through)
	if gameID != "" {
		query = query.Where("(game_id IS NULL OR game_id = '' OR game_id = ?)", gameID)
	}
	if campaignID != "" {
		query = query.Where("(campaign_id IS NULL OR campaign_id = '' OR campaign_id = ?)", campaignID)
	}
	var rows []researchdomain.Source
	err := query.Order("published_at DESC, created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
