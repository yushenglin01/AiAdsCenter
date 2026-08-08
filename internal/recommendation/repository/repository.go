package repository

import (
	"context"
	"errors"

	businessdomain "github.com/example/adnova/internal/business/domain"
	"github.com/example/adnova/internal/common/apperror"
	recommendationdomain "github.com/example/adnova/internal/recommendation/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, tenantID string, filter recommendationdomain.Filter) ([]recommendationdomain.Detail, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.RiskLevel != "" {
		query = query.Where("risk_level = ?", filter.RiskLevel)
	}
	var recommendations []businessdomain.Recommendation
	if err := query.Order("created_at DESC").Limit(limit).Find(&recommendations).Error; err != nil {
		return nil, err
	}
	rows := make([]recommendationdomain.Detail, 0, len(recommendations))
	for i := range recommendations {
		detail, err := r.build(ctx, tenantID, recommendations[i])
		if err != nil {
			return nil, err
		}
		rows = append(rows, *detail)
	}
	return rows, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, id string) (*recommendationdomain.Detail, error) {
	var recommendation businessdomain.Recommendation
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&recommendation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound
		}
		return nil, err
	}
	return r.build(ctx, tenantID, recommendation)
}

func (r *Repository) build(ctx context.Context, tenantID string, recommendation businessdomain.Recommendation) (*recommendationdomain.Detail, error) {
	result := &recommendationdomain.Detail{Recommendation: recommendation}
	var approval businessdomain.ApprovalRequest
	q := r.db.WithContext(ctx).Where("tenant_id = ? AND recommendation_id = ?", tenantID, recommendation.ID).Limit(1).Find(&approval)
	if q.Error != nil {
		return nil, q.Error
	}
	if q.RowsAffected == 1 {
		result.Approval = &approval
		result.ReportID = approval.ReportID
	}
	var campaign struct{ Name string }
	if err := r.db.WithContext(ctx).Table("campaigns").Select("name").Where("tenant_id = ? AND id = ?", tenantID, recommendation.CampaignID).Scan(&campaign).Error; err != nil {
		return nil, err
	}
	result.CampaignName = campaign.Name
	var task struct{ Status string }
	if err := r.db.WithContext(ctx).Table("agent_tasks").Select("status").Where("tenant_id = ? AND id = ?", tenantID, recommendation.TaskID).Scan(&task).Error; err != nil {
		return nil, err
	}
	result.TaskStatus = task.Status
	return result, nil
}
