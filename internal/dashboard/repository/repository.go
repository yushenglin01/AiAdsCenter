package repository

import (
	"context"

	dashboarddomain "github.com/example/adnova/internal/dashboard/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Summary(ctx context.Context, tenantID string) (*dashboarddomain.OperationsSummary, error) {
	result := &dashboarddomain.OperationsSummary{}
	counts := []struct {
		table  string
		where  string
		args   []any
		target *int64
	}{
		{"approval_requests", "tenant_id = ? AND status = ?", []any{tenantID, "PENDING"}, &result.PendingApprovals},
		{"agent_tasks", "tenant_id = ?", []any{tenantID}, &result.TotalTasks},
		{"agent_tasks", "tenant_id = ? AND status = ?", []any{tenantID, "SUCCEEDED"}, &result.SucceededTasks},
		{"agent_tasks", "tenant_id = ? AND status IN ?", []any{tenantID, []string{"FAILED", "MANUAL_REVIEW", "CANCELLED"}}, &result.FailedTasks},
		{"agent_tasks", "tenant_id = ? AND status IN ?", []any{tenantID, []string{"PENDING", "RUNNING", "RETRYING"}}, &result.ActiveTasks},
		{"model_usage_records", "tenant_id = ?", []any{tenantID}, &result.ModelCalls},
		{"audit_logs", "tenant_id = ?", []any{tenantID}, &result.AuditEvents},
	}
	for _, count := range counts {
		if err := r.db.WithContext(ctx).Table(count.table).Where(count.where, count.args...).Count(count.target).Error; err != nil {
			return nil, err
		}
	}
	if result.SucceededTasks+result.FailedTasks > 0 {
		result.TaskSuccessRate = float64(result.SucceededTasks) / float64(result.SucceededTasks+result.FailedTasks)
	}
	type costRow struct{ Cost decimal.Decimal }
	var cost costRow
	if err := r.db.WithContext(ctx).Table("model_usage_records").Where("tenant_id = ?", tenantID).Select("COALESCE(SUM(estimated_cost),0) AS cost").Scan(&cost).Error; err != nil {
		return nil, err
	}
	result.ModelCost = cost.Cost
	return result, nil
}
