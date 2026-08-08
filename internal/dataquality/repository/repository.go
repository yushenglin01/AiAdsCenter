package repository

import (
	"context"
	"fmt"
	"time"

	dataqualitydomain "github.com/example/adnova/internal/dataquality/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

type aggregateRow struct {
	RowCount      int64      `gorm:"column:row_count"`
	CampaignCount int64      `gorm:"column:campaign_count"`
	LatestDate    *time.Time `gorm:"column:latest_date"`
}

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) SourceSnapshot(ctx context.Context, tenantID, gameID, source string) (dataqualitydomain.SourceSnapshot, error) {
	table, ok := map[string]string{
		"AD":           "normalized_ad_metrics",
		"MMP":          "mmp_metrics",
		"GAME_REVENUE": "game_revenue_metrics",
		"CREATIVE":     "creative_daily_metrics",
	}[source]
	if !ok {
		return dataqualitydomain.SourceSnapshot{}, fmt.Errorf("unsupported data quality source %s", source)
	}
	var row aggregateRow
	err := r.db.WithContext(ctx).Table(table).
		Select("COUNT(*) AS row_count, COUNT(DISTINCT campaign_id) AS campaign_count, MAX(date) AS latest_date").
		Where("tenant_id = ? AND game_id = ?", tenantID, gameID).
		Scan(&row).Error
	if err != nil {
		return dataqualitydomain.SourceSnapshot{}, err
	}
	return dataqualitydomain.SourceSnapshot{Name: source, RowCount: row.RowCount, CampaignCount: row.CampaignCount, LatestDate: row.LatestDate}, nil
}

func (r *Repository) ImportSummary(ctx context.Context, tenantID, gameID string) (dataqualitydomain.ImportSummary, error) {
	var rows []struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:count"`
	}
	err := r.db.WithContext(ctx).Table("data_import_jobs").
		Select("status, COUNT(*) AS count").
		Where("tenant_id = ? AND game_id = ?", tenantID, gameID).
		Group("status").Scan(&rows).Error
	if err != nil {
		return dataqualitydomain.ImportSummary{}, err
	}
	var result dataqualitydomain.ImportSummary
	for _, row := range rows {
		switch row.Status {
		case "SUCCEEDED":
			result.Succeeded = row.Count
		case "FAILED":
			result.Failed = row.Count
		}
	}
	return result, nil
}
