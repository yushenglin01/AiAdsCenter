package repository

import (
	"context"

	creativedomain "github.com/example/adnova/internal/creative/domain"
	ingestiondomain "github.com/example/adnova/internal/ingestion/domain"
	rulesdomain "github.com/example/adnova/internal/rules/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) LoadMetrics(ctx context.Context, tenantID, gameID string) ([]ingestiondomain.CreativeDailyMetric, error) {
	var rows []ingestiondomain.CreativeDailyMetric
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Order("creative_id, date").Find(&rows).Error
	return rows, err
}

func (r *Repository) LoadCreatives(ctx context.Context, tenantID string) ([]creativedomain.Creative, error) {
	var rows []creativedomain.Creative
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&rows).Error
	return rows, err
}

func (r *Repository) LoadRules(ctx context.Context, tenantID string) ([]rulesdomain.AnalysisRule, error) {
	var rows []rulesdomain.AnalysisRule
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND category = ? AND enabled = ?", tenantID, "CREATIVE", true).Find(&rows).Error
	return rows, err
}

func (r *Repository) Replace(ctx context.Context, tenantID, gameID string, rows []creativedomain.Finding) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Delete(&creativedomain.Finding{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.CreateInBatches(rows, 100).Error
	})
}

func (r *Repository) List(ctx context.Context, tenantID, gameID string) ([]creativedomain.Finding, error) {
	var rows []creativedomain.Finding
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if gameID != "" {
		query = query.Where("game_id = ?", gameID)
	}
	err := query.Order("FIELD(severity, 'CRITICAL', 'HIGH', 'MEDIUM', 'LOW'), fatigue_score DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) Get(ctx context.Context, tenantID, id string) (*creativedomain.Finding, error) {
	var row creativedomain.Finding
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	return &row, err
}
