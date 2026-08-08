package repository

import (
	"context"
	"errors"
	"time"

	"github.com/example/adnova/internal/mmp/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("mmp resource not found")

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) GameExists(ctx context.Context, tenantID, gameID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("games").Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, gameID).Count(&count).Error
	return count == 1, err
}

func (r *Repository) ListConnections(ctx context.Context, tenantID string) ([]domain.Connection, error) {
	var rows []domain.Connection
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) GetConnection(ctx context.Context, tenantID, id string) (*domain.Connection, error) {
	var row domain.Connection
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (r *Repository) FindConnection(ctx context.Context, tenantID, gameID, provider string) (*domain.Connection, error) {
	var row domain.Connection
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND game_id = ? AND provider = ?", tenantID, gameID, provider).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (r *Repository) UpsertConnection(ctx context.Context, row *domain.Connection) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "game_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"external_app_id", "status", "updated_by", "updated_at"}),
	}).Create(row).Error
}

func (r *Repository) TouchConnection(ctx context.Context, tenantID, id string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&domain.Connection{}).Where("tenant_id = ? AND id = ?", tenantID, id).Update("last_sync_at", at).Error
}

func (r *Repository) ListSyncRuns(ctx context.Context, tenantID string) ([]domain.SyncRun, error) {
	var rows []domain.SyncRun
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(200).Find(&rows).Error
	return rows, err
}

func (r *Repository) FindSyncRunByKey(ctx context.Context, tenantID, key string) (*domain.SyncRun, error) {
	var row domain.SyncRun
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (r *Repository) CreateSyncRun(ctx context.Context, row *domain.SyncRun) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateSyncRun(ctx context.Context, row *domain.SyncRun) error {
	return r.db.WithContext(ctx).Save(row).Error
}
