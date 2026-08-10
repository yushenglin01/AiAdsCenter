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
	Intent  string `json:"intent,omitempty"`
	Input   Input  `json:"input,omitempty"`
	Message string `json:"message,omitempty"`
	Confirm bool   `json:"confirm,omitempty"`
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
	parser        Parser
}

func New(workflow Workflow, approvals Approvals, notifications Notifications, parsers ...Parser) *Service {
	result := &Service{workflow: workflow, approvals: approvals, notifications: notifications}
	if len(parsers) > 0 {
		result.parser = parsers[0]
	}
	return result
}

func (s *Service) Execute(ctx context.Context, actor Actor, command Command) (*Result, error) {
	parsedByLLM := false
	var parsed *ParsedCommand
	if command.Intent == "" {
		if command.Message == "" {
			return nil, fmt.Errorf("intent or message is required")
		}
		if s.parser == nil || !s.parser.Enabled() {
			return nil, fmt.Errorf("OpenClaw natural-language parsing is not configured")
		}
		var err error
		parsed, err = s.parser.Parse(ctx, actor, command.Message)
		if err != nil {
			return nil, err
		}
		command.Intent, command.Input, parsedByLLM = parsed.Intent, parsed.Input, true
		if parsed.RequiresConfirmation && !command.Confirm {
			return &Result{Intent: parsed.Intent, Status: "NEEDS_CONFIRMATION", Data: map[string]any{"parsed_command": parsed, "executed": false}}, nil
		}
	}
	switch command.Intent {
	case IntentRunFullAnalysis:
		row, err := s.workflow.Start(ctx, actor.TenantID, actor.UserID, actor.TraceID, workflowservice.StartInput{GameID: command.Input.GameID, CampaignID: command.Input.CampaignID, AnalysisDate: command.Input.AnalysisDate})
		if err != nil {
			return nil, err
		}
		return commandResult(command.Intent, "ACCEPTED", true, row, parsedByLLM, parsed), nil
	case IntentGetWorkflowStatus:
		if command.Input.WorkflowID == "" {
			return nil, fmt.Errorf("workflow_id is required")
		}
		row, err := s.workflow.Get(ctx, actor.TenantID, command.Input.WorkflowID)
		if err != nil {
			return nil, err
		}
		return commandResult(command.Intent, "SUCCEEDED", false, row, parsedByLLM, parsed), nil
	case IntentListPendingApprovals:
		rows, err := s.approvals.List(ctx, actor.TenantID, approvaldomain.Filter{Status: approvaldomain.StatusPending, Limit: 50})
		if err != nil {
			return nil, err
		}
		return commandResult(command.Intent, "SUCCEEDED", false, rows, parsedByLLM, parsed), nil
	case IntentListNotifications:
		rows, err := s.notifications.List(ctx, actor.TenantID, command.Input.NotificationStatus)
		if err != nil {
			return nil, err
		}
		return commandResult(command.Intent, "SUCCEEDED", false, rows, parsedByLLM, parsed), nil
	case IntentMarkNotificationRead:
		if command.Input.NotificationID == "" {
			return nil, fmt.Errorf("notification_id is required")
		}
		row, err := s.notifications.MarkRead(ctx, actor.TenantID, command.Input.NotificationID, actor.UserID)
		if err != nil {
			return nil, err
		}
		return commandResult(command.Intent, "SUCCEEDED", false, row, parsedByLLM, parsed), nil
	default:
		return nil, fmt.Errorf("unsupported OpenClaw intent %s", command.Intent)
	}
}

func commandResult(intent, status string, accepted bool, data any, parsedByLLM bool, parsed *ParsedCommand) *Result {
	if !parsedByLLM {
		return &Result{Intent: intent, Status: status, Accepted: accepted, Data: data}
	}
	return &Result{Intent: intent, Status: status, Accepted: accepted, Data: map[string]any{"result": data, "parsed_command": parsed, "executed": true}}
}
