package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	agentruntime "github.com/example/adnova/internal/agent/runtime"
	businessrepo "github.com/example/adnova/internal/business/repository"
	notificationservice "github.com/example/adnova/internal/notification/service"
	"github.com/example/adnova/internal/workflow/domain"
	"github.com/example/adnova/internal/workflow/repository"
	"github.com/google/uuid"
)

const WorkflowFullAnalysis = "FULL_ANALYSIS"

type StartInput struct {
	GameID       string `json:"game_id"`
	CampaignID   string `json:"campaign_id"`
	AnalysisDate string `json:"analysis_date,omitempty"`
}

type BusinessReader interface {
	Get(ctx context.Context, tenantID, taskID string) (*businessrepo.TaskDetails, error)
}

type Notifier interface {
	Publish(ctx context.Context, input notificationservice.PublishInput) error
}

type Service struct {
	repo     *repository.Repository
	registry *agentruntime.Registry
	business BusinessReader
	notifier Notifier
}

func New(repo *repository.Repository, registry *agentruntime.Registry, business BusinessReader, notifiers ...Notifier) *Service {
	result := &Service{repo: repo, registry: registry, business: business}
	if len(notifiers) > 0 {
		result.notifier = notifiers[0]
	}
	return result
}

func (s *Service) Start(ctx context.Context, tenantID, userID, traceID string, input StartInput) (*domain.Details, error) {
	if input.GameID == "" || input.CampaignID == "" {
		return nil, fmt.Errorf("game_id and campaign_id are required")
	}
	if input.AnalysisDate == "" {
		input.AnalysisDate = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", input.AnalysisDate); err != nil {
		return nil, fmt.Errorf("analysis_date must be YYYY-MM-DD")
	}
	idempotencyKey := fmt.Sprintf("%s:%s:%s:%s", WorkflowFullAnalysis, input.GameID, input.CampaignID, input.AnalysisDate)
	existing, err := s.repo.FindByIdempotency(ctx, tenantID, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return s.Get(ctx, tenantID, existing.ID)
	}
	now := time.Now().UTC()
	payload, _ := json.Marshal(input)
	run := &domain.Run{ID: uuid.NewString(), TenantID: tenantID, WorkflowType: WorkflowFullAnalysis, GameID: input.GameID, CampaignID: input.CampaignID, Status: "RUNNING", CurrentStep: "openclaw-agent", IdempotencyKey: idempotencyKey, InputJSON: payload, TriggeredBy: userID, TraceID: traceID, StartedAt: &now}
	steps := newSteps(run.ID, tenantID)
	if err := s.repo.Create(ctx, run, steps); err != nil {
		// The database unique key closes the race between concurrent retries.
		// If another request won, return that durable workflow instead of a
		// duplicate-key error.
		existing, findErr := s.repo.FindByIdempotency(ctx, tenantID, idempotencyKey)
		if findErr == nil && existing != nil {
			return s.Get(ctx, tenantID, existing.ID)
		}
		return nil, err
	}
	agentInput := agentdomain.AgentInput{WorkflowID: run.ID, TenantID: tenantID, UserID: userID, TraceID: traceID, Payload: payload}
	agentInput.TaskID = stepID(steps, "openclaw-agent")
	if _, err := s.executeStep(ctx, "openclaw-agent", agentInput); err != nil {
		_ = s.finishRun(ctx, tenantID, run.ID, "FAILED", "openclaw-agent", err.Error(), nil)
		return s.repo.Get(ctx, tenantID, run.ID)
	}
	for _, name := range []string{"data-agent", "attribution-agent", "creative-agent", "research-agent"} {
		agentInput.TaskID = stepID(steps, name)
		if _, err := s.executeStep(ctx, name, agentInput); err != nil {
			_ = s.finishRun(ctx, tenantID, run.ID, "FAILED", name, err.Error(), nil)
			return s.repo.Get(ctx, tenantID, run.ID)
		}
	}
	agentInput.TaskID = stepID(steps, "business-agent")
	result, err := s.executeStep(ctx, "business-agent", agentInput)
	if err != nil {
		_ = s.finishRun(ctx, tenantID, run.ID, "FAILED", "business-agent", err.Error(), nil)
		return s.repo.Get(ctx, tenantID, run.ID)
	}
	if result.ExternalTaskID == "" {
		err = fmt.Errorf("business-agent did not return an external task id")
		_ = s.finishRun(ctx, tenantID, run.ID, "FAILED", "business-agent", err.Error(), nil)
		return s.repo.Get(ctx, tenantID, run.ID)
	}
	if err := s.repo.LinkBusinessTask(ctx, tenantID, run.ID, result.ExternalTaskID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, tenantID, run.ID)
}

func (s *Service) executeStep(ctx context.Context, name string, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	if err := s.repo.StartStep(ctx, input.TenantID, input.WorkflowID, name, input.Payload); err != nil {
		return nil, err
	}
	result, err := s.registry.Execute(ctx, name, input)
	if err != nil {
		_ = s.repo.FinishStep(ctx, input.TenantID, input.WorkflowID, name, "FAILED", "", nil, err.Error())
		return nil, err
	}
	if err := s.repo.FinishStep(ctx, input.TenantID, input.WorkflowID, name, result.Status, result.ExternalTaskID, result.Output, ""); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, tenantID, workflowID string) (*domain.Details, error) {
	details, err := s.repo.Get(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	if details.Run.BusinessTaskID == "" || workflowTerminal(details.Run.Status) {
		return details, nil
	}
	task, err := s.business.Get(ctx, tenantID, details.Run.BusinessTaskID)
	if err != nil {
		return nil, err
	}
	if err := s.reconcileBusiness(ctx, details, task); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, tenantID, workflowID)
}

func (s *Service) reconcileBusiness(ctx context.Context, details *domain.Details, task *businessrepo.TaskDetails) error {
	tenantID, workflowID := details.Run.TenantID, details.Run.ID
	status := task.Task.Status
	switch status {
	case "PENDING", "RUNNING", "RETRYING":
		return s.repo.FinishStep(ctx, tenantID, workflowID, "business-agent", status, task.Task.ID, task.Task.OutputJSON, task.Task.ErrorMessage)
	case "SUCCEEDED", "WAITING_APPROVAL":
		if err := s.repo.FinishStep(ctx, tenantID, workflowID, "business-agent", status, task.Task.ID, task.Task.OutputJSON, ""); err != nil {
			return err
		}
		if !stepFinished(details.Steps, "report-agent") {
			payload, _ := json.Marshal(map[string]any{"game_id": details.Run.GameID, "campaign_id": details.Run.CampaignID, "business_task_id": task.Task.ID})
			input := agentdomain.AgentInput{WorkflowID: workflowID, TaskID: stepID(details.Steps, "report-agent"), TenantID: tenantID, UserID: details.Run.TriggeredBy, TraceID: details.Run.TraceID, Payload: payload}
			if _, err := s.executeStep(ctx, "report-agent", input); err != nil {
				_ = s.finishRun(ctx, tenantID, workflowID, "FAILED", "report-agent", err.Error(), nil)
				return err
			}
		}
		if status == "WAITING_APPROVAL" {
			return s.finishRun(ctx, tenantID, workflowID, "WAITING_APPROVAL", "human-approval", "", task.Task.OutputJSON)
		}
		return s.finishRun(ctx, tenantID, workflowID, "COMPLETED", "completed", "", task.Task.OutputJSON)
	case "MANUAL_REVIEW":
		_ = s.repo.FinishStep(ctx, tenantID, workflowID, "business-agent", status, task.Task.ID, task.Task.OutputJSON, task.Task.ErrorMessage)
		return s.finishRun(ctx, tenantID, workflowID, "MANUAL_REVIEW", "business-agent", task.Task.ErrorMessage, task.Task.OutputJSON)
	case "FAILED", "CANCELLED":
		_ = s.repo.FinishStep(ctx, tenantID, workflowID, "business-agent", status, task.Task.ID, task.Task.OutputJSON, task.Task.ErrorMessage)
		return s.finishRun(ctx, tenantID, workflowID, status, "business-agent", task.Task.ErrorMessage, task.Task.OutputJSON)
	default:
		return fmt.Errorf("unsupported business task status %s", status)
	}
}

func (s *Service) finishRun(ctx context.Context, tenantID, workflowID, status, currentStep, message string, output json.RawMessage) error {
	if err := s.repo.FinishRun(ctx, tenantID, workflowID, status, currentStep, message, output); err != nil {
		return err
	}
	if s.notifier == nil {
		return nil
	}
	title, notificationMessage := notificationContent(status)
	if title == "" {
		return nil
	}
	return s.notifier.Publish(ctx, notificationservice.PublishInput{TenantID: tenantID, WorkflowID: workflowID, EventType: "WORKFLOW_" + status, Title: title, Message: notificationMessage, Metadata: map[string]any{"workflow_status": status, "current_step": currentStep, "external_delivery": "NOT_CONFIGURED"}})
}

func notificationContent(status string) (string, string) {
	switch status {
	case "COMPLETED":
		return "多 Agent 分析已完成", "经营分析报告已经生成，可查看结论和建议。"
	case "WAITING_APPROVAL":
		return "经营建议等待人工审批", "高风险建议已进入审批中心，尚未执行任何广告平台操作。"
	case "FAILED":
		return "多 Agent 分析失败", "工作流已停止，请查看失败步骤后重试。"
	case "MANUAL_REVIEW":
		return "模型结果需要人工复核", "Business Agent 输出未通过结构化校验。"
	case "CANCELLED":
		return "多 Agent 分析已取消", "工作流已取消，没有执行广告平台操作。"
	default:
		return "", ""
	}
}

func (s *Service) List(ctx context.Context, tenantID string) ([]domain.Run, error) {
	return s.repo.List(ctx, tenantID, 50)
}

func newSteps(workflowID, tenantID string) []domain.Step {
	definitions := []struct{ name, mode string }{
		{"openclaw-agent", "INTERACTION_GATEWAY"},
		{"data-agent", "SYNCHRONOUS_DETERMINISTIC"},
		{"attribution-agent", "SYNCHRONOUS_DETERMINISTIC"},
		{"creative-agent", "SYNCHRONOUS_DETERMINISTIC"},
		{"research-agent", "EXTERNAL_RESEARCH"},
		{"business-agent", "ASYNCHRONOUS_LLM"},
		{"report-agent", "SYNCHRONOUS_DETERMINISTIC"},
	}
	steps := make([]domain.Step, 0, len(definitions))
	for index, definition := range definitions {
		steps = append(steps, domain.Step{ID: uuid.NewString(), TenantID: tenantID, WorkflowID: workflowID, AgentName: definition.name, SequenceNumber: index + 1, Status: "PENDING", ExecutionMode: definition.mode})
	}
	return steps
}

func stepID(steps []domain.Step, name string) string {
	for _, step := range steps {
		if step.AgentName == name {
			return step.ID
		}
	}
	return ""
}

func stepFinished(steps []domain.Step, name string) bool {
	for _, step := range steps {
		if step.AgentName == name {
			return step.Status == "SUCCEEDED" || step.Status == "SKIPPED"
		}
	}
	return false
}

func workflowTerminal(status string) bool {
	return status == "COMPLETED" || status == "FAILED" || status == "MANUAL_REVIEW" || status == "CANCELLED"
}
