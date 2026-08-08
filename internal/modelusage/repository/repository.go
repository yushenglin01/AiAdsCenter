package repository

import (
	"context"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	modelusagedomain "github.com/example/adnova/internal/modelusage/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, tenantID string, limit int) ([]agentdomain.ModelUsageRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var rows []agentdomain.ModelUsageRecord
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) Summary(ctx context.Context, tenantID string) (*modelusagedomain.Summary, error) {
	type aggregate struct {
		Calls           int64
		SuccessfulCalls int64
		InputTokens     int64
		OutputTokens    int64
		EstimatedCost   decimal.Decimal
		AverageLatency  float64
	}
	var total aggregate
	if err := r.db.WithContext(ctx).Table("model_usage_records").Where("tenant_id = ?", tenantID).
		Select("COUNT(*) AS calls, SUM(CASE WHEN status = 'SUCCEEDED' THEN 1 ELSE 0 END) AS successful_calls, COALESCE(SUM(input_tokens),0) AS input_tokens, COALESCE(SUM(output_tokens),0) AS output_tokens, COALESCE(SUM(estimated_cost),0) AS estimated_cost, COALESCE(AVG(latency_ms),0) AS average_latency").Scan(&total).Error; err != nil {
		return nil, err
	}
	result := &modelusagedomain.Summary{Calls: total.Calls, SuccessfulCalls: total.SuccessfulCalls, InputTokens: total.InputTokens, OutputTokens: total.OutputTokens, EstimatedCost: total.EstimatedCost, AverageLatency: total.AverageLatency}
	if result.Calls > 0 {
		result.SuccessRate = float64(result.SuccessfulCalls) / float64(result.Calls)
	}
	if err := r.db.WithContext(ctx).Table("model_usage_records").Where("tenant_id = ?", tenantID).Group("provider").Order("calls DESC").
		Select("provider, COUNT(*) AS calls, COALESCE(SUM(input_tokens),0) AS input_tokens, COALESCE(SUM(output_tokens),0) AS output_tokens, COALESCE(SUM(estimated_cost),0) AS estimated_cost, COALESCE(AVG(latency_ms),0) AS average_latency").Scan(&result.Providers).Error; err != nil {
		return nil, err
	}
	return result, nil
}
