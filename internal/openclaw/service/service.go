package service

import (
	"context"
	"fmt"

	approvaldomain "github.com/example/adnova/internal/approval/domain"
	notificationdomain "github.com/example/adnova/internal/notification/domain"
	workflowdomain "github.com/example/adnova/internal/workflow/domain"
	workflowservice "github.com/example/adnova/internal/workflow/service"
)

const (
	IntentRunFullAnalysis      = "RUN_FULL_ANALYSIS"
	IntentGetWorkflowStatus    = "GET_WORKFLOW_STATUS"
	IntentListPendingApprovals = "LIST_PENDING_APPROVALS"
	IntentListNotifications    = "LIST_NOTIFICATIONS"
	IntentMarkNotificationRead = "MARK_NOTIFICATION_READ"
)

type Workflow interface {
	Start(context.Context, string, string, string, workflowservice.StartInput) (*workflowdomain.Details, error)
	Get(context.Context, string, string) (*workflowdomain.Details, error)
}

type Approvals interface {
	List(context.Context, string, approvaldomain.Filter) ([]approvaldomain.Detail, error)
}

type Notifications interface {
	List(context.Context, string, string) ([]notificationdomain.Notification, error)
	MarkRead(context.Context, string, string, string) (*notificationdomain.Notification, error)
}

type Input struct {
	GameID             string `json:"game_id,omitempty"`
	CampaignID         string `json:"campaign_id,omitempty"`
	AnalysisDate       string `json:"analysis_date,omitempty"`
	WorkflowID         string `json:"workflow_id,omitempty"`
	NotificationStatus string `json:"notification_status,omitempty"`
	NotificationID     string `json:"notification_id,omitempty"`
}

type Command struct {
	Intent string `json:"intent"`
	Input  Input  `json:"input"`
}

type Actor struct {
	TenantID string
	UserID   string
	TraceID  string
}

type Result struct {
	Intent   string `json:"intent"`
	Status   string `json:"status"`
	Accepted bool   `json:"-"`
	Data     any    `json:"data"`
}

type Service struct {
	workflow      Workflow
	approvals     Approvals
	notifications Notifications
}

func New(workflow Workflow, approvals Approvals, notifications Notifications) *Service {
	return &Service{workflow: workflow, approvals: approvals, notifications: notifications}
}

func (s *Service) Execute(ctx context.Context, actor Actor, command Command) (*Result, error) {
	switch command.Intent {
	case IntentRunFullAnalysis:
		row, err := s.workflow.Start(ctx, actor.TenantID, actor.UserID, actor.TraceID, workflowservice.StartInput{GameID: command.Input.GameID, CampaignID: command.Input.CampaignID, AnalysisDate: command.Input.AnalysisDate})
		if err != nil {
			return nil, err
		}
		return &Result{Intent: command.Intent, Status: "ACCEPTED", Accepted: true, Data: row}, nil
	case IntentGetWorkflowStatus:
		if command.Input.WorkflowID == "" {
			return nil, fmt.Errorf("workflow_id is required")
		}
		row, err := s.workflow.Get(ctx, actor.TenantID, command.Input.WorkflowID)
		if err != nil {
			return nil, err
		}
		return &Result{Intent: command.Intent, Status: "SUCCEEDED", Data: row}, nil
	case IntentListPendingApprovals:
		rows, err := s.approvals.List(ctx, actor.TenantID, approvaldomain.Filter{Status: approvaldomain.StatusPending, Limit: 50})
		if err != nil {
			return nil, err
		}
		return &Result{Intent: command.Intent, Status: "SUCCEEDED", Data: rows}, nil
	case IntentListNotifications:
		rows, err := s.notifications.List(ctx, actor.TenantID, command.Input.NotificationStatus)
		if err != nil {
			return nil, err
		}
		return &Result{Intent: command.Intent, Status: "SUCCEEDED", Data: rows}, nil
	case IntentMarkNotificationRead:
		if command.Input.NotificationID == "" {
			return nil, fmt.Errorf("notification_id is required")
		}
		row, err := s.notifications.MarkRead(ctx, actor.TenantID, command.Input.NotificationID, actor.UserID)
		if err != nil {
			return nil, err
		}
		return &Result{Intent: command.Intent, Status: "SUCCEEDED", Data: row}, nil
	default:
		return nil, fmt.Errorf("unsupported OpenClaw intent %s", command.Intent)
	}
}
