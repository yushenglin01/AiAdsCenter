package repository

import (
	"context"

	"github.com/example/adnova/internal/rules/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListRules(ctx context.Context, tenantID string) ([]domain.AnalysisRule, error) {
	var rows []domain.AnalysisRule
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("category, code").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListBenchmarks(ctx context.Context, tenantID, gameID string) ([]domain.BusinessBenchmark, error) {
	var rows []domain.BusinessBenchmark
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if gameID != "" {
		query = query.Where("game_id = ?", gameID)
	}
	err := query.Order("metric_code").Find(&rows).Error
	return rows, err
}

func (r *Repository) UpdateRule(ctx context.Context, tenantID, id string, values map[string]any) (*domain.AnalysisRule, error) {
	result := r.db.WithContext(ctx).Model(&domain.AnalysisRule{}).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(values)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row domain.AnalysisRule
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) ReplaceFindings(ctx context.Context, tenantID, gameID string, rows []domain.RuleFinding) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Delete(&domain.RuleFinding{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.CreateInBatches(rows, 100).Error
	})
}

func (r *Repository) ListFindings(ctx context.Context, tenantID, gameID string) ([]domain.RuleFinding, error) {
	var rows []domain.RuleFinding
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if gameID != "" {
		query = query.Where("game_id = ?", gameID)
	}
	err := query.Order("FIELD(severity, 'CRITICAL', 'HIGH', 'MEDIUM', 'LOW'), created_at DESC").Find(&rows).Error
	return rows, err
}
