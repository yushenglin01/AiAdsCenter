package repository

import (
	"context"
	"errors"
	"time"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/notification/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateOnce(ctx context.Context, row *domain.Notification) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenant_id"}, {Name: "workflow_id"}, {Name: "event_type"}}, DoNothing: true}).Create(row).Error
}

func (r *Repository) List(ctx context.Context, tenantID, status string, limit int) ([]domain.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var rows []domain.Notification
	err := query.Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) Get(ctx context.Context, tenantID, id string) (*domain.Notification, error) {
	var row domain.Notification
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *Repository) MarkRead(ctx context.Context, tenantID, id, userID string) (*domain.Notification, error) {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&domain.Notification{}).Where("tenant_id = ? AND id = ? AND status = ?", tenantID, id, domain.StatusUnread).Updates(map[string]any{"status": domain.StatusRead, "read_by": userID, "read_at": now})
	if result.Error != nil {
		return nil, result.Error
	}
	return r.Get(ctx, tenantID, id)
}
