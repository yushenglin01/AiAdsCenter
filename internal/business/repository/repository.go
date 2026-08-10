package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	"github.com/example/adnova/internal/business/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

type TaskDetails struct {
	Task            agentdomain.AgentTask          `json:"task"`
	Attempts        []agentdomain.AgentTaskAttempt `json:"attempts"`
	Usage           []agentdomain.ModelUsageRecord `json:"usage"`
	Findings        []domain.BusinessFinding       `json:"findings"`
	Recommendations []domain.Recommendation        `json:"recommendations"`
	Approvals       []domain.ApprovalRequest       `json:"approvals"`
	Report          *domain.AnalysisReport         `json:"report,omitempty"`
}

type workflowStepOutput struct {
	AgentName string          `gorm:"column:agent_name"`
	Output    json.RawMessage `gorm:"column:output_json"`
}

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindTaskByIdempotency(ctx context.Context, tenantID, key string) (*agentdomain.AgentTask, error) {
	var task agentdomain.AgentTask
	result := r.db.WithContext(ctx).Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).Limit(1).Find(&task)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &task, nil
}

func (r *Repository) FindTaskSystemByID(ctx context.Context, taskID string) (*agentdomain.AgentTask, error) {
	var task agentdomain.AgentTask
	if err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *Repository) CreateTask(ctx context.Context, task *agentdomain.AgentTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *Repository) CreateTaskWithOutbox(ctx context.Context, task *agentdomain.AgentTask, outbox *agentdomain.TaskOutbox) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		return tx.Create(outbox).Error
	})
}

func (r *Repository) PendingOutbox(ctx context.Context, limit int) ([]agentdomain.TaskOutbox, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []agentdomain.TaskOutbox
	err := r.db.WithContext(ctx).Where("status = ? AND next_attempt_at <= ?", "PENDING", time.Now().UTC()).Order("created_at ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&agentdomain.TaskOutbox{}).Where("id = ? AND status = ?", id, "PENDING").Updates(map[string]any{"status": "PUBLISHED", "published_at": now, "last_error": "", "dispatch_count": gorm.Expr("dispatch_count + 1")}).Error
}

func (r *Repository) MarkOutboxRetry(ctx context.Context, id string, err error) error {
	next := time.Now().UTC().Add(5 * time.Second)
	return r.db.WithContext(ctx).Model(&agentdomain.TaskOutbox{}).Where("id = ? AND status = ?", id, "PENDING").Updates(map[string]any{"last_error": err.Error(), "next_attempt_at": next, "dispatch_count": gorm.Expr("dispatch_count + 1")}).Error
}

func (r *Repository) MarkTaskQueued(ctx context.Context, tenantID, taskID string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&agentdomain.AgentTask{}).Where("tenant_id = ? AND id = ? AND status = ?", tenantID, taskID, "PENDING").Updates(map[string]any{"queued_at": now, "current_step": "QUEUED"}).Error
}

func (r *Repository) ClaimTask(ctx context.Context, tenantID, taskID string, retryCount int, staleBefore time.Time) (*agentdomain.AgentTask, error) {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&agentdomain.AgentTask{}).
		Where("tenant_id = ? AND id = ? AND (status IN ? OR (status = ? AND (last_heartbeat_at IS NULL OR last_heartbeat_at < ?)))", tenantID, taskID, []string{"PENDING", "RETRYING"}, "RUNNING", staleBefore).
		Updates(map[string]any{"status": "RUNNING", "current_step": "COLLECTING_CONTEXT", "queue_retry_count": retryCount, "started_at": now, "last_heartbeat_at": now, "error_message": ""})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, errors.New("agent task is not claimable")
	}
	return r.FindTaskSystemByID(ctx, taskID)
}

func (r *Repository) HeartbeatTask(ctx context.Context, tenantID, taskID string) error {
	return r.db.WithContext(ctx).Model(&agentdomain.AgentTask{}).Where("tenant_id = ? AND id = ? AND status = ?", tenantID, taskID, "RUNNING").Updates(map[string]any{"last_heartbeat_at": time.Now().UTC()}).Error
}

func (r *Repository) MarkTaskRetrying(ctx context.Context, tenantID, taskID, message string, retryCount int) error {
	result := r.db.WithContext(ctx).Model(&agentdomain.AgentTask{}).Where("tenant_id = ? AND id = ? AND status = ?", tenantID, taskID, "RUNNING").Updates(map[string]any{"status": "RETRYING", "current_step": "RETRY_SCHEDULED", "error_message": message, "queue_retry_count": retryCount, "last_heartbeat_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("agent task retry transition rejected")
	}
	return nil
}

func (r *Repository) MarkTaskFailed(ctx context.Context, tenantID, taskID, message string, retryCount int) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&agentdomain.AgentTask{}).Where("tenant_id = ? AND id = ? AND status IN ?", tenantID, taskID, []string{"RUNNING", "RETRYING"}).Updates(map[string]any{"status": "FAILED", "current_step": "RETRY_EXHAUSTED", "error_message": message, "queue_retry_count": retryCount, "finished_at": now, "last_heartbeat_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("agent task failure transition rejected")
	}
	return nil
}

func (r *Repository) DeleteGeneratedResults(ctx context.Context, tenantID, taskID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, value := range []any{&domain.ApprovalRequest{}, &domain.Recommendation{}, &domain.BusinessFinding{}, &domain.AnalysisReport{}} {
			if err := tx.Where("tenant_id = ? AND task_id = ?", tenantID, taskID).Delete(value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) SetTaskInput(ctx context.Context, tenantID, taskID string, input json.RawMessage) error {
	return r.db.WithContext(ctx).Model(&agentdomain.AgentTask{}).Where("tenant_id = ? AND id = ?", tenantID, taskID).Update("input_json", input).Error
}

func (r *Repository) SetTaskAttempt(ctx context.Context, tenantID, taskID string, attempt int) error {
	return r.db.WithContext(ctx).Model(&agentdomain.AgentTask{}).Where("tenant_id = ? AND id = ?", tenantID, taskID).Update("attempt", attempt).Error
}

func (r *Repository) TransitionTask(ctx context.Context, tenantID, taskID, from, to, step, errorMessage string, output json.RawMessage) error {
	values := map[string]any{"status": to, "current_step": step, "error_message": errorMessage}
	now := time.Now().UTC()
	if to == "RUNNING" {
		values["started_at"] = now
	}
	if to == "SUCCEEDED" || to == "FAILED" || to == "MANUAL_REVIEW" || to == "CANCELLED" {
		values["finished_at"] = now
	}
	if len(output) > 0 {
		values["output_json"] = output
	}
	result := r.db.WithContext(ctx).Model(&agentdomain.AgentTask{}).Where("tenant_id = ? AND id = ? AND status = ?", tenantID, taskID, from).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("illegal or concurrent agent task transition")
	}
	return nil
}

func (r *Repository) CreateAttempt(ctx context.Context, row *agentdomain.AgentTaskAttempt) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) CreateUsage(ctx context.Context, row *agentdomain.ModelUsageRecord) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) WorkflowAgentOutputs(ctx context.Context, tenantID, workflowID string, agentNames []string) (map[string]json.RawMessage, error) {
	result := map[string]json.RawMessage{}
	if workflowID == "" || len(agentNames) == 0 {
		return result, nil
	}
	var rows []workflowStepOutput
	err := r.db.WithContext(ctx).Table("workflow_steps").
		Select("agent_name, output_json").
		Where("tenant_id = ? AND workflow_id = ? AND agent_name IN ? AND status IN ?", tenantID, workflowID, agentNames, []string{"SUCCEEDED", "SKIPPED"}).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if len(row.Output) > 0 && json.Valid(row.Output) {
			result[row.AgentName] = append(json.RawMessage(nil), row.Output...)
		}
	}
	return result, nil
}

func (r *Repository) CreateFinding(ctx context.Context, row *domain.BusinessFinding) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) CreateRecommendation(ctx context.Context, row *domain.Recommendation) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) GetRecommendation(ctx context.Context, tenantID, id string) (*domain.Recommendation, error) {
	var row domain.Recommendation
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CreateApproval(ctx context.Context, row *domain.ApprovalRequest) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) CreateReport(ctx context.Context, row *domain.AnalysisReport) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) GetTask(ctx context.Context, tenantID, taskID string) (*TaskDetails, error) {
	result := &TaskDetails{}
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, taskID).First(&result.Task).Error; err != nil {
		return nil, err
	}
	queries := []struct {
		value any
		order string
	}{
		{&result.Attempts, "attempt_number ASC"}, {&result.Usage, "created_at ASC"}, {&result.Findings, "created_at ASC"}, {&result.Recommendations, "priority ASC"}, {&result.Approvals, "created_at ASC"},
	}
	for _, query := range queries {
		if err := r.db.WithContext(ctx).Where("tenant_id = ? AND task_id = ?", tenantID, taskID).Order(query.order).Find(query.value).Error; err != nil {
			return nil, err
		}
	}
	var report domain.AnalysisReport
	reportQuery := r.db.WithContext(ctx).Where("tenant_id = ? AND task_id = ?", tenantID, taskID).Limit(1).Find(&report)
	if reportQuery.Error != nil {
		return nil, reportQuery.Error
	}
	if reportQuery.RowsAffected == 1 {
		result.Report = &report
	}
	return result, nil
}

func (r *Repository) GetReport(ctx context.Context, tenantID, taskID string) (*domain.AnalysisReport, error) {
	var report domain.AnalysisReport
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND task_id = ?", tenantID, taskID).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *Repository) ListTasks(ctx context.Context, tenantID string, limit int) ([]agentdomain.AgentTask, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []agentdomain.AgentTask
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND task_type = ?", tenantID, "BUSINESS_ANALYSIS").Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
