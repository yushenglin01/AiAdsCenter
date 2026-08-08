package repository

import (
	"context"

	campaigndomain "github.com/example/adnova/internal/campaign/domain"
	ingestiondomain "github.com/example/adnova/internal/ingestion/domain"
	"github.com/example/adnova/internal/metrics/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }
func (r *Repository) LoadAds(ctx context.Context, tenantID, gameID string) ([]ingestiondomain.NormalizedAdMetric, error) {
	var rows []ingestiondomain.NormalizedAdMetric
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
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if gameID != "" {
		q = q.Where("game_id = ?", gameID)
	}
	err := q.Find(&rows).Error
	return rows, err
}
func (r *Repository) Replace(ctx context.Context, tenantID, gameID string, rows []domain.CampaignDailyMetric) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND game_id = ?", tenantID, gameID).Delete(&domain.CampaignDailyMetric{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(rows, 200).Error
	})
}
func (r *Repository) ListDaily(ctx context.Context, tenantID, gameID, campaignID string) ([]domain.CampaignDailyMetric, error) {
	var rows []domain.CampaignDailyMetric
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if gameID != "" {
		q = q.Where("game_id = ?", gameID)
	}
	if campaignID != "" {
		q = q.Where("campaign_id = ?", campaignID)
	}
	err := q.Order("date ASC").Find(&rows).Error
	return rows, err
}
func (r *Repository) RiskCounts(ctx context.Context, tenantID string) (int64, int64, int64, error) {
	var business, attribution, creative int64
	if err := r.db.WithContext(ctx).Table("rule_findings").Where("tenant_id = ? AND severity IN ?", tenantID, []string{"HIGH", "CRITICAL"}).Distinct("campaign_id").Count(&business).Error; err != nil {
		return 0, 0, 0, err
	}
	if err := r.db.WithContext(ctx).Table("attribution_findings").Where("tenant_id = ? AND severity IN ?", tenantID, []string{"HIGH", "CRITICAL"}).Count(&attribution).Error; err != nil {
		return 0, 0, 0, err
	}
	if err := r.db.WithContext(ctx).Table("creative_findings").Where("tenant_id = ? AND severity IN ?", tenantID, []string{"HIGH", "CRITICAL"}).Distinct("creative_id").Count(&creative).Error; err != nil {
		return 0, 0, 0, err
	}
	return business, attribution, creative, nil
}
