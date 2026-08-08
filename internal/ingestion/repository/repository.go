package repository

import (
	"context"
	"errors"
	"time"

	campaigndomain "github.com/example/adnova/internal/campaign/domain"
	creativedomain "github.com/example/adnova/internal/creative/domain"
	gamedomain "github.com/example/adnova/internal/game/domain"
	"github.com/example/adnova/internal/ingestion/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("import resource not found")

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) GameExists(ctx context.Context, tenantID, gameID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("games").Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, gameID).Count(&count).Error
	return count == 1, err
}
func (r *Repository) FindGameByCode(ctx context.Context, tenantID, code string) (*gamedomain.Game, error) {
	var row gamedomain.Game
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND code = ? AND deleted_at IS NULL", tenantID, code).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) FindCampaign(ctx context.Context, tenantID, externalID string) (*campaigndomain.Campaign, error) {
	var row campaigndomain.Campaign
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND external_id = ?", tenantID, externalID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) FindCreative(ctx context.Context, tenantID, externalID string) (*creativedomain.Creative, error) {
	var row creativedomain.Creative
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND external_id = ?", tenantID, externalID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) FindJobByKey(ctx context.Context, tenantID, key string) (*domain.ImportJob, error) {
	var row domain.ImportJob
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}
func (r *Repository) CreateJob(ctx context.Context, job *domain.ImportJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}
func (r *Repository) UpdateJob(ctx context.Context, job *domain.ImportJob) error {
	return r.db.WithContext(ctx).Save(job).Error
}
func (r *Repository) CountJobRows(ctx context.Context, importType, jobID string) (int, error) {
	tables := map[string]string{
		"AD_METRICS": "normalized_ad_metrics", "MMP_METRICS": "mmp_metrics",
		"GAME_REVENUE": "game_revenue_metrics", "CREATIVE_METRICS": "creative_daily_metrics",
	}
	table, ok := tables[importType]
	if !ok {
		return 0, ErrNotFound
	}
	var count int64
	if err := r.db.WithContext(ctx).Table(table).Where("import_job_id = ?", jobID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}
func (r *Repository) ListJobs(ctx context.Context, tenantID string) ([]domain.ImportJob, error) {
	var rows []domain.ImportJob
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(200).Find(&rows).Error
	return rows, err
}
func (r *Repository) GetJob(ctx context.Context, tenantID, id string) (*domain.ImportJob, error) {
	var row domain.ImportJob
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (r *Repository) InsertAd(ctx context.Context, raw []domain.RawAdMetric, normalized []domain.NormalizedAdMetric) (int, error) {
	imported := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range raw {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&raw[i]).Error; err != nil {
				return err
			}
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&normalized[i])
			if result.Error != nil {
				return result.Error
			}
			imported += int(result.RowsAffected)
		}
		return nil
	})
	return imported, err
}
func (r *Repository) InsertMMP(ctx context.Context, rows []domain.MMPMetric) (int, error) {
	return insertRows(ctx, r.db, rows)
}
func (r *Repository) ReplaceMMP(ctx context.Context, tenantID, source, gameID string, periodStart, periodEnd time.Time, rows []domain.MMPMetric) (int, error) {
	imported := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND source = ? AND game_id = ? AND date BETWEEN ? AND ?", tenantID, source, gameID, periodStart, periodEnd).Delete(&domain.MMPMetric{}).Error; err != nil {
			return err
		}
		for i := range rows {
			if err := tx.Create(&rows[i]).Error; err != nil {
				return err
			}
			imported++
		}
		return nil
	})
	return imported, err
}
func (r *Repository) InsertRevenue(ctx context.Context, rows []domain.GameRevenueMetric) (int, error) {
	return insertRows(ctx, r.db, rows)
}
func (r *Repository) InsertCreative(ctx context.Context, rows []domain.CreativeDailyMetric) (int, error) {
	return insertRows(ctx, r.db, rows)
}

func insertRows[T any](ctx context.Context, db *gorm.DB, rows []T) (int, error) {
	imported := 0
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range rows {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows[i])
			if result.Error != nil {
				return result.Error
			}
			imported += int(result.RowsAffected)
		}
		return nil
	})
	return imported, err
}
