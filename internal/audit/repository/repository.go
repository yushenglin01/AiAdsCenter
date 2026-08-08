package repository

import (
	"context"

	"github.com/example/adnova/internal/audit/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, row *domain.AuditLog) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) List(ctx context.Context, tenantID string, filter domain.Filter) ([]domain.AuditLog, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ActorID != "" {
		query = query.Where("actor_id = ?", filter.ActorID)
	}
	if filter.TaskID != "" {
		query = query.Where("task_id = ?", filter.TaskID)
	}
	var rows []domain.AuditLog
	err := query.Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
