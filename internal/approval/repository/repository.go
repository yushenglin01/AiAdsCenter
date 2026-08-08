package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	approvaldomain "github.com/example/adnova/internal/approval/domain"
	auditdomain "github.com/example/adnova/internal/audit/domain"
	businessdomain "github.com/example/adnova/internal/business/domain"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, tenantID string, filter approvaldomain.Filter) ([]approvaldomain.Detail, error) {
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
	var approvals []businessdomain.ApprovalRequest
	if err := query.Order("created_at DESC").Limit(limit).Find(&approvals).Error; err != nil {
		return nil, err
	}
	rows := make([]approvaldomain.Detail, 0, len(approvals))
	for i := range approvals {
		detail, err := r.getWithDB(ctx, r.db, tenantID, approvals[i].ID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, *detail)
	}
	return rows, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, approvalID string) (*approvaldomain.Detail, error) {
	return r.getWithDB(ctx, r.db, tenantID, approvalID)
}

func (r *Repository) getWithDB(ctx context.Context, db *gorm.DB, tenantID, approvalID string) (*approvaldomain.Detail, error) {
	result := &approvaldomain.Detail{}
	if err := db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, approvalID).First(&result.Approval).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound
		}
		return nil, err
	}
	if err := db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, result.Approval.RecommendationID).First(&result.Recommendation).Error; err != nil {
		return nil, err
	}
	if err := db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, result.Approval.TaskID).First(&result.Task).Error; err != nil {
		return nil, err
	}
	var report businessdomain.AnalysisReport
	q := db.WithContext(ctx).Where("tenant_id = ? AND task_id = ?", tenantID, result.Approval.TaskID).Limit(1).Find(&report)
	if q.Error != nil {
		return nil, q.Error
	}
	if q.RowsAffected == 1 {
		result.Report = &report
	}
	var campaign struct{ Name string }
	if err := db.WithContext(ctx).Table("campaigns").Select("name").Where("tenant_id = ? AND id = ?", tenantID, result.Recommendation.CampaignID).Scan(&campaign).Error; err != nil {
		return nil, err
	}
	result.CampaignName = campaign.Name
	return result, nil
}

func (r *Repository) Decide(ctx context.Context, decision approvaldomain.Decision) (*approvaldomain.Detail, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var approval businessdomain.ApprovalRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", decision.TenantID, decision.ApprovalID).First(&approval).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.NotFound
			}
			return err
		}
		if approval.Status != approvaldomain.StatusPending {
			return apperror.Conflict
		}
		var task agentdomain.AgentTask
		if err := tx.Where("tenant_id = ? AND id = ?", decision.TenantID, approval.TaskID).First(&task).Error; err != nil {
			return err
		}
		if decision.ActorID == identity.SystemAgentUserID || decision.ActorID == approval.RequestedBy || decision.ActorID == task.CreatedBy {
			return apperror.Forbidden
		}
		var recommendation businessdomain.Recommendation
		if err := tx.Where("tenant_id = ? AND id = ?", decision.TenantID, approval.RecommendationID).First(&recommendation).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		before, _ := json.Marshal(approval)
		updates := map[string]any{"status": decision.Status, "reviewed_by": decision.ActorID, "review_comment": decision.Comment, "reviewed_at": now, "decision_version": gorm.Expr("decision_version + 1")}
		changed := tx.Model(&businessdomain.ApprovalRequest{}).Where("tenant_id = ? AND id = ? AND status = ?", decision.TenantID, decision.ApprovalID, approvaldomain.StatusPending).Updates(updates)
		if changed.Error != nil {
			return changed.Error
		}
		if changed.RowsAffected != 1 {
			return apperror.Conflict
		}
		recommendationStatus := decision.Status
		changed = tx.Model(&businessdomain.Recommendation{}).Where("tenant_id = ? AND id = ? AND status = ?", decision.TenantID, recommendation.ID, "PROPOSED").Update("status", recommendationStatus)
		if changed.Error != nil {
			return changed.Error
		}
		if changed.RowsAffected != 1 {
			return apperror.Conflict
		}
		var pending int64
		if err := tx.Model(&businessdomain.ApprovalRequest{}).Where("tenant_id = ? AND task_id = ? AND status = ?", decision.TenantID, approval.TaskID, approvaldomain.StatusPending).Count(&pending).Error; err != nil {
			return err
		}
		if pending == 0 {
			if err := tx.Model(&agentdomain.AgentTask{}).Where("tenant_id = ? AND id = ? AND status = ?", decision.TenantID, approval.TaskID, "WAITING_APPROVAL").Updates(map[string]any{"status": "SUCCEEDED", "current_step": "COMPLETED", "finished_at": now}).Error; err != nil {
				return err
			}
		}
		approval.Status, approval.DecidedBy, approval.DecisionComment, approval.DecidedAt = decision.Status, decision.ActorID, decision.Comment, &now
		approval.DecisionVersion++
		after, _ := json.Marshal(approval)
		metadata, _ := json.Marshal(map[string]any{"recommendation_id": recommendation.ID, "recommendation_status": recommendationStatus, "remaining_pending_approvals": pending, "advertising_platform_called": false})
		audit := &auditdomain.AuditLog{ID: uuid.NewString(), TenantID: decision.TenantID, ActorID: decision.ActorID, ActorType: decision.ActorType, Action: "APPROVAL_" + decision.Status, ResourceType: "APPROVAL_REQUEST", ResourceID: approval.ID, TaskID: approval.TaskID, BeforeJSON: before, AfterJSON: after, MetadataJSON: metadata, RequestID: decision.RequestID, TraceID: decision.TraceID, IPAddress: decision.IPAddress}
		return tx.Create(audit).Error
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, decision.TenantID, decision.ApprovalID)
}
