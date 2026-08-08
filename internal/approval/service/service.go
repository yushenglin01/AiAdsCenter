package service

import (
	"context"
	"errors"
	"strings"

	approvaldomain "github.com/example/adnova/internal/approval/domain"
	auditdomain "github.com/example/adnova/internal/audit/domain"
	auditservice "github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/common/apperror"
)

type Repository interface {
	List(context.Context, string, approvaldomain.Filter) ([]approvaldomain.Detail, error)
	Get(context.Context, string, string) (*approvaldomain.Detail, error)
	Decide(context.Context, approvaldomain.Decision) (*approvaldomain.Detail, error)
}

type Service struct {
	repo    Repository
	auditor auditservice.Recorder
}

func New(repo Repository, auditors ...auditservice.Recorder) *Service {
	result := &Service{repo: repo}
	if len(auditors) > 0 {
		result.auditor = auditors[0]
	}
	return result
}

func (s *Service) List(ctx context.Context, tenantID string, filter approvaldomain.Filter) ([]approvaldomain.Detail, error) {
	return s.repo.List(ctx, tenantID, filter)
}

func (s *Service) Get(ctx context.Context, tenantID, approvalID string) (*approvaldomain.Detail, error) {
	return s.repo.Get(ctx, tenantID, approvalID)
}

func (s *Service) Decide(ctx context.Context, decision approvaldomain.Decision, roles []string) (*approvaldomain.Detail, error) {
	if !hasRole(roles, "ADMIN", "MANAGER") || hasRole(roles, "SYSTEM_AGENT") {
		if err := s.recordFailure(ctx, decision, "APPROVAL_DECISION_DENIED", apperror.Forbidden); err != nil {
			return nil, err
		}
		return nil, apperror.Forbidden
	}
	if decision.Status != approvaldomain.StatusApproved && decision.Status != approvaldomain.StatusRejected {
		err := apperror.Validation("approval decision must be APPROVED or REJECTED")
		if auditErr := s.recordFailure(ctx, decision, "APPROVAL_DECISION_DENIED", err); auditErr != nil {
			return nil, auditErr
		}
		return nil, err
	}
	decision.Comment = strings.TrimSpace(decision.Comment)
	if decision.Status == approvaldomain.StatusRejected && decision.Comment == "" {
		err := apperror.Validation("rejection comment is required")
		if auditErr := s.recordFailure(ctx, decision, "APPROVAL_DECISION_DENIED", err); auditErr != nil {
			return nil, auditErr
		}
		return nil, err
	}
	result, err := s.repo.Decide(ctx, decision)
	if err != nil {
		if auditErr := s.recordFailure(ctx, decision, "APPROVAL_DECISION_FAILED", err); auditErr != nil {
			return nil, auditErr
		}
		return nil, err
	}
	return result, nil
}

func (s *Service) recordFailure(ctx context.Context, decision approvaldomain.Decision, action string, decisionErr error) error {
	if s.auditor == nil {
		return nil
	}
	reason := apperror.Internal.Message
	var appErr *apperror.Error
	if errors.As(decisionErr, &appErr) {
		reason = appErr.Message
	}
	if decision.TaskID == "" {
		if detail, err := s.repo.Get(ctx, decision.TenantID, decision.ApprovalID); err == nil {
			decision.TaskID = detail.Approval.TaskID
		}
	}
	return s.auditor.Record(ctx, auditdomain.RecordInput{
		TenantID: decision.TenantID, ActorID: decision.ActorID, ActorType: decision.ActorType,
		Action: action, ResourceType: "APPROVAL_REQUEST", ResourceID: decision.ApprovalID, TaskID: decision.TaskID,
		Metadata: map[string]any{"requested_status": decision.Status, "reason": reason}, RequestID: decision.RequestID, TraceID: decision.TraceID, IPAddress: decision.IPAddress,
	})
}

func hasRole(actual []string, expected ...string) bool {
	for _, role := range actual {
		for _, value := range expected {
			if role == value {
				return true
			}
		}
	}
	return false
}
