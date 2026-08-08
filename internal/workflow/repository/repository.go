package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/example/adnova/internal/workflow/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, run *domain.Run, steps []domain.Step) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(run).Error; err != nil {
			return err
		}
		return tx.Create(&steps).Error
	})
}

func (r *Repository) FindByIdempotency(ctx context.Context, tenantID, key string) (*domain.Run, error) {
	var run domain.Run
	result := r.db.WithContext(ctx).Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).Limit(1).Find(&run)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &run, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, workflowID string) (*domain.Details, error) {
	result := &domain.Details{}
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, workflowID).First(&result.Run).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND workflow_id = ?", tenantID, workflowID).Order("sequence_number ASC").Find(&result.Steps).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) List(ctx context.Context, tenantID string, limit int) ([]domain.Run, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []domain.Run
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *Repository) StartStep(ctx context.Context, tenantID, workflowID, agentName string, input json.RawMessage) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&domain.Step{}).Where("tenant_id = ? AND workflow_id = ? AND agent_name = ? AND status = ?", tenantID, workflowID, agentName, "PENDING").Updates(map[string]any{"status": "RUNNING", "input_json": input, "started_at": now, "error_message": ""})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("workflow step is not startable")
		}
		return tx.Model(&domain.Run{}).Where("tenant_id = ? AND id = ?", tenantID, workflowID).Updates(map[string]any{"status": "RUNNING", "current_step": agentName, "error_message": ""}).Error
	})
}

func (r *Repository) FinishStep(ctx context.Context, tenantID, workflowID, agentName, status, externalTaskID string, output json.RawMessage, message string) error {
	values := map[string]any{"status": status, "external_task_id": externalTaskID, "output_json": output, "error_message": message}
	if terminalStep(status) {
		values["finished_at"] = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Model(&domain.Step{}).Where("tenant_id = ? AND workflow_id = ? AND agent_name = ?", tenantID, workflowID, agentName).Updates(values).Error
}

func (r *Repository) LinkBusinessTask(ctx context.Context, tenantID, workflowID, taskID string) error {
	return r.db.WithContext(ctx).Model(&domain.Run{}).Where("tenant_id = ? AND id = ?", tenantID, workflowID).Updates(map[string]any{"business_task_id": taskID, "status": "WAITING_AGENT", "current_step": "business-agent"}).Error
}

func (r *Repository) FinishRun(ctx context.Context, tenantID, workflowID, status, currentStep, message string, output json.RawMessage) error {
	values := map[string]any{"status": status, "current_step": currentStep, "error_message": message, "output_json": output}
	if terminalRun(status) {
		values["finished_at"] = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Model(&domain.Run{}).Where("tenant_id = ? AND id = ?", tenantID, workflowID).Updates(values).Error
}

func terminalStep(status string) bool {
	return status == "SUCCEEDED" || status == "SKIPPED" || status == "FAILED" || status == "MANUAL_REVIEW" || status == "CANCELLED" || status == "WAITING_APPROVAL"
}

func terminalRun(status string) bool {
	return status == "COMPLETED" || status == "FAILED" || status == "MANUAL_REVIEW" || status == "CANCELLED"
}
