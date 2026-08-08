package repository

import (
	"context"

	"github.com/example/adnova/internal/attribution/domain"
	campaigndomain "github.com/example/adnova/internal/campaign/domain"
	ingestiondomain "github.com/example/adnova/internal/ingestion/domain"
	rulesdomain "github.com/example/adnova/internal/rules/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) LoadAds(ctx context.Context, tenantID, gameID string) ([]ingestiondomain.NormalizedAdMetric, error) {
	var rows []ingestiondomain.NormalizedAdMetric
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Find(&rows).Error
	return rows, err
}

func (r *Repository) LoadMMP(ctx context.Context, tenantID, gameID string) ([]ingestiondomain.MMPMetric, error) {
	var rows []ingestiondomain.MMPMetric
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Find(&rows).Error
	return rows, err
}

func (r *Repository) LoadRevenue(ctx context.Context, tenantID, gameID string) ([]ingestiondomain.GameRevenueMetric, error) {
	var rows []ingestiondomain.GameRevenueMetric
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Find(&rows).Error
	return rows, err
}

func (r *Repository) LoadCampaigns(ctx context.Context, tenantID, gameID string) ([]campaigndomain.Campaign, error) {
	var rows []campaigndomain.Campaign
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Find(&rows).Error
	return rows, err
}

func (r *Repository) LoadRules(ctx context.Context, tenantID string) ([]rulesdomain.AnalysisRule, error) {
	var rows []rulesdomain.AnalysisRule
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND category = ? AND enabled = ?", tenantID, "ATTRIBUTION", true).Find(&rows).Error
	return rows, err
}

func (r *Repository) Replace(ctx context.Context, tenantID, gameID string, rows []domain.Finding) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Delete(&domain.Finding{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.CreateInBatches(rows, 100).Error
	})
}

func (r *Repository) List(ctx context.Context, tenantID, gameID string) ([]domain.Finding, error) {
	var rows []domain.Finding
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if gameID != "" {
		query = query.Where("game_id = ?", gameID)
	}
	err := query.Order("FIELD(severity, 'CRITICAL', 'HIGH', 'MEDIUM', 'LOW'), created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) Get(ctx context.Context, tenantID, id string) (*domain.Finding, error) {
	var row domain.Finding
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	return &row, err
}
