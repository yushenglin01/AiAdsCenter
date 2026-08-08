package domain

import "github.com/shopspring/decimal"

type OperationsSummary struct {
	PendingApprovals int64           `json:"pending_approvals"`
	TotalTasks       int64           `json:"total_tasks"`
	SucceededTasks   int64           `json:"succeeded_tasks"`
	FailedTasks      int64           `json:"failed_tasks"`
	ActiveTasks      int64           `json:"active_tasks"`
	TaskSuccessRate  float64         `json:"task_success_rate"`
	ModelCalls       int64           `json:"model_calls"`
	ModelCost        decimal.Decimal `json:"model_cost"`
	AuditEvents      int64           `json:"audit_events"`
}
