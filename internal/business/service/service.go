package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	agenttools "github.com/example/adnova/internal/agent/tools"
	attributiondomain "github.com/example/adnova/internal/attribution/domain"
	attributionrepo "github.com/example/adnova/internal/attribution/repository"
	auditservice "github.com/example/adnova/internal/audit/service"
	"github.com/example/adnova/internal/business/domain"
	"github.com/example/adnova/internal/business/repository"
	creativeanalysisrepo "github.com/example/adnova/internal/creative/analysis/repository"
	creativedomain "github.com/example/adnova/internal/creative/domain"
	"github.com/example/adnova/internal/llm"
	metricsdto "github.com/example/adnova/internal/metrics/dto"
	metricsservice "github.com/example/adnova/internal/metrics/service"
	reportservice "github.com/example/adnova/internal/report/service"
	researchdomain "github.com/example/adnova/internal/research/domain"
	researchrepo "github.com/example/adnova/internal/research/repository"
	rulesdomain "github.com/example/adnova/internal/rules/domain"
	rulesrepo "github.com/example/adnova/internal/rules/repository"
	"github.com/example/adnova/internal/taskqueue"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AnalyzeInput struct {
	GameID       string `json:"game_id"`
	CampaignID   string `json:"campaign_id"`
	AnalysisDate string `json:"analysis_date"`
}

type BusinessInput struct {
	TaskID              string                          `json:"task_id"`
	TenantID            string                          `json:"tenant_id"`
	GameID              string                          `json:"game_id"`
	Campaign            metricsdto.CampaignMetric       `json:"campaign"`
	Benchmarks          []rulesdomain.BusinessBenchmark `json:"benchmark"`
	AttributionFindings []attributiondomain.Finding     `json:"attribution_findings"`
	CreativeFindings    []creativedomain.Finding        `json:"creative_findings"`
	RuleFindings        []rulesdomain.RuleFinding       `json:"rule_findings"`
	ResearchSources     []researchdomain.Evidence       `json:"research_sources"`
	Forecast            map[string]any                  `json:"forecast"`
	Constraints         []string                        `json:"constraints"`
}

type Service struct {
	repo        *repository.Repository
	metrics     *metricsservice.Service
	rules       *rulesrepo.Repository
	attr        *attributionrepo.Repository
	creative    *creativeanalysisrepo.Repository
	research    *researchrepo.Repository
	registry    *agenttools.Registry
	client      llm.Client
	validator   *Validator
	auditor     auditservice.Recorder
	prompts     PromptSet
	spec        agentdomain.AgentSpec
	enqueuer    taskqueue.Enqueuer
	taskTimeout time.Duration
}

func New(repo *repository.Repository, metrics *metricsservice.Service, rules *rulesrepo.Repository, attr *attributionrepo.Repository, creative *creativeanalysisrepo.Repository, research *researchrepo.Repository, client llm.Client, prompts PromptSet, model string, enqueuer taskqueue.Enqueuer, taskTimeout time.Duration, auditor auditservice.Recorder) (*Service, error) {
	s := &Service{repo: repo, metrics: metrics, rules: rules, attr: attr, creative: creative, research: research, registry: agenttools.NewRegistry(), client: client, validator: NewValidator(), auditor: auditor, prompts: prompts, enqueuer: enqueuer, taskTimeout: taskTimeout}
	s.spec = agentdomain.AgentSpec{Name: "business-agent", Version: "1.1.0", Description: "海外游戏广告经营风险分析", ExecutionMode: "ASYNCHRONOUS_LLM", Model: model, SystemPrompt: prompts.BusinessSystem, Tools: []string{"get_campaign_metrics", "get_business_benchmark", "get_attribution_anomalies", "get_creative_findings", "get_verified_research_sources", "forecast_ltv", "create_recommendation", "create_approval_request"}, Permissions: []string{"READ_METRICS", "READ_ANALYSIS", "READ_VERIFIED_RESEARCH", "CREATE_RECOMMENDATION", "CREATE_APPROVAL_REQUEST"}, MaxSteps: 10, Timeout: 30 * time.Second, InputSchema: "business-agent-input/1.1.0", OutputSchema: prompts.OutputSchema}
	if err := s.registerTools(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Submit(ctx context.Context, tenantID, userID, traceID string, input AnalyzeInput) (*repository.TaskDetails, error) {
	return s.submit(ctx, tenantID, userID, traceID, "", input)
}

func (s *Service) SubmitForWorkflow(ctx context.Context, tenantID, userID, traceID, workflowID string, input AnalyzeInput) (*repository.TaskDetails, error) {
	if workflowID == "" {
		return nil, fmt.Errorf("workflow_id is required")
	}
	return s.submit(ctx, tenantID, userID, traceID, workflowID, input)
}

func (s *Service) submit(ctx context.Context, tenantID, userID, traceID, workflowID string, input AnalyzeInput) (*repository.TaskDetails, error) {
	if input.GameID == "" || input.CampaignID == "" {
		return nil, fmt.Errorf("game_id and campaign_id are required")
	}
	analysisDate := input.AnalysisDate
	if analysisDate == "" {
		analysisDate = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", analysisDate); err != nil {
		return nil, fmt.Errorf("analysis_date must be YYYY-MM-DD")
	}
	idempotencyKey := fmt.Sprintf("%s:%s:%s:%s:BUSINESS_ANALYSIS", tenantID, input.GameID, input.CampaignID, analysisDate)
	if workflowID != "" {
		idempotencyKey += ":" + workflowID
	}
	existing, err := s.repo.FindTaskByIdempotency(ctx, tenantID, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		_ = s.DispatchPending(ctx, 10)
		return s.repo.GetTask(ctx, tenantID, existing.ID)
	}
	requestJSON, _ := json.Marshal(input)
	now := time.Now().UTC()
	task := &agentdomain.AgentTask{ID: uuid.NewString(), TenantID: tenantID, WorkflowID: workflowID, AgentName: "business-agent", GameID: input.GameID, CampaignID: input.CampaignID, TaskType: "BUSINESS_ANALYSIS", SchemaName: s.prompts.SchemaName, SchemaVersion: s.prompts.SchemaVersion, Status: "PENDING", IdempotencyKey: idempotencyKey, InputJSON: requestJSON, CurrentStep: "OUTBOX_PENDING", MaxAttempts: 2, ScheduledAt: now, CreatedBy: userID}
	payload := taskqueue.BusinessAnalysisPayload{TaskID: task.ID, TenantID: tenantID, GameID: input.GameID, CampaignID: input.CampaignID, CreatedBy: userID, AnalysisDate: analysisDate, TraceID: traceID}
	payloadJSON, _ := json.Marshal(payload)
	outbox := &agentdomain.TaskOutbox{ID: uuid.NewString(), TenantID: tenantID, TaskID: task.ID, TaskType: taskqueue.TypeBusinessAnalysis, PayloadJSON: payloadJSON, Status: "PENDING", NextAttemptAt: now}
	if err := s.repo.CreateTaskWithOutbox(ctx, task, outbox); err != nil {
		return nil, err
	}
	_ = s.dispatchOutbox(ctx, *outbox)
	return s.repo.GetTask(ctx, tenantID, task.ID)
}

func (s *Service) DispatchPending(ctx context.Context, limit int) error {
	if s.enqueuer == nil {
		return fmt.Errorf("task queue enqueuer is not configured")
	}
	rows, err := s.repo.PendingOutbox(ctx, limit)
	if err != nil {
		return err
	}
	var failures []string
	for _, row := range rows {
		if err := s.dispatchOutbox(ctx, row); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("dispatch outbox: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (s *Service) dispatchOutbox(ctx context.Context, row agentdomain.TaskOutbox) error {
	if s.enqueuer == nil {
		return fmt.Errorf("task queue enqueuer is not configured")
	}
	var payload taskqueue.BusinessAnalysisPayload
	if err := json.Unmarshal(row.PayloadJSON, &payload); err != nil {
		_ = s.repo.MarkOutboxRetry(ctx, row.ID, err)
		return err
	}
	if err := s.enqueuer.EnqueueBusinessAnalysis(ctx, payload); err != nil {
		_ = s.repo.MarkOutboxRetry(ctx, row.ID, err)
		return err
	}
	if err := s.repo.MarkOutboxPublished(ctx, row.ID); err != nil {
		return err
	}
	return s.repo.MarkTaskQueued(ctx, payload.TenantID, payload.TaskID)
}

func (s *Service) ProcessQueued(ctx context.Context, payload taskqueue.BusinessAnalysisPayload, queueRetry int) error {
	existing, err := s.repo.FindTaskSystemByID(ctx, payload.TaskID)
	if err != nil {
		return err
	}
	if existing.TenantID != payload.TenantID || existing.GameID != payload.GameID || existing.CampaignID != payload.CampaignID || existing.CreatedBy != payload.CreatedBy {
		return fmt.Errorf("queued task identity does not match persisted task")
	}
	if terminalStatus(existing.Status) {
		return nil
	}
	timeout := s.taskTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	task, err := s.repo.ClaimTask(ctx, payload.TenantID, payload.TaskID, queueRetry, time.Now().UTC().Add(-timeout))
	if err != nil {
		return err
	}
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	go s.heartbeat(heartbeatCtx, payload.TenantID, payload.TaskID)
	if err := s.repo.DeleteGeneratedResults(ctx, payload.TenantID, payload.TaskID); err != nil {
		return err
	}
	businessInput, err := s.collectInput(ctx, payload.TenantID, payload.CreatedBy, payload.TaskID, payload.GameID, payload.CampaignID, payload.AnalysisDate)
	if err != nil {
		return fmt.Errorf("collect business context: %w", err)
	}
	structuredInput, _ := json.Marshal(businessInput)
	if err := s.repo.SetTaskInput(ctx, payload.TenantID, payload.TaskID, structuredInput); err != nil {
		return err
	}
	validated, raw, err := s.generateAndValidate(ctx, payload.TenantID, payload.TaskID, payload.TraceID, structuredInput, task.MaxAttempts, queueRetry)
	if err != nil {
		return err
	}
	if validated == nil {
		return s.repo.TransitionTask(ctx, payload.TenantID, payload.TaskID, "RUNNING", "MANUAL_REVIEW", "VALIDATION_FAILED", "model output failed validation twice", raw)
	}
	report := reportservice.Compose(reportservice.Input{TenantID: payload.TenantID, TaskID: payload.TaskID, WorkflowID: task.WorkflowID, GameID: payload.GameID, CampaignID: payload.CampaignID, Result: *validated, Research: businessInput.ResearchSources, AnalysisDate: payload.AnalysisDate})
	if err := s.repo.CreateReport(ctx, report); err != nil {
		return fmt.Errorf("create analysis report: %w", err)
	}
	if err := s.persistValidated(ctx, payload.TenantID, payload.TaskID, payload.GameID, payload.CampaignID, report.ID, *validated); err != nil {
		return fmt.Errorf("persist business result: %w", err)
	}
	output, _ := json.Marshal(validated)
	status, step := "SUCCEEDED", "COMPLETED"
	for _, recommendation := range validated.Recommendations {
		if recommendation.RequiresApproval {
			status, step = "WAITING_APPROVAL", "APPROVAL_CREATION"
			break
		}
	}
	return s.repo.TransitionTask(ctx, payload.TenantID, payload.TaskID, "RUNNING", status, step, "", output)
}

func (s *Service) heartbeat(ctx context.Context, tenantID, taskID string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.repo.HeartbeatTask(ctx, tenantID, taskID)
		}
	}
}

func terminalStatus(status string) bool {
	return status == "WAITING_APPROVAL" || status == "SUCCEEDED" || status == "FAILED" || status == "MANUAL_REVIEW" || status == "CANCELLED"
}

func (s *Service) generateAndValidate(ctx context.Context, tenantID, taskID, traceID string, input json.RawMessage, maxAttempts, queueRetry int) (*domain.Result, json.RawMessage, error) {
	var raw json.RawMessage
	var previousErrors []string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptNumber := queueRetry*maxAttempts + attempt
		_ = s.repo.SetTaskAttempt(ctx, tenantID, taskID, attemptNumber)
		started := time.Now().UTC()
		userPrompt := s.prompts.BusinessUser
		if len(previousErrors) > 0 {
			userPrompt += "\n\n上一次输出被校验器拒绝，请修正以下问题后只返回完整 JSON：\n"
			for _, validationError := range previousErrors {
				userPrompt += "- " + validationError + "\n"
			}
		}
		response, err := s.client.GenerateStructured(ctx, llm.GenerateRequest{Model: s.spec.Model, SystemPrompt: s.spec.SystemPrompt, UserPrompt: userPrompt, InputJSON: input, OutputSchema: s.spec.OutputSchema, Temperature: 0.1, MaxTokens: 2000, Timeout: s.spec.Timeout, TraceID: traceID})
		row := &agentdomain.AgentTaskAttempt{ID: uuid.NewString(), TenantID: tenantID, TaskID: taskID, AttemptNumber: attemptNumber, Provider: s.client.Name(), Model: s.spec.Model, StartedAt: started, FinishedAt: time.Now().UTC()}
		if err != nil {
			row.Status, row.ErrorMessage = "FAILED", err.Error()
			_ = s.repo.CreateAttempt(ctx, row)
			return nil, nil, fmt.Errorf("generate structured business result: %w", err)
		}
		raw, row.Model, row.RawResponseJSON = response.Content, response.Model, response.Content
		result, validationErrors := s.validator.Validate(response.Content, input)
		usageStatus := "SUCCEEDED"
		if len(validationErrors) > 0 {
			previousErrors = validationErrors
			usageStatus, row.Status = "VALIDATION_FAILED", "VALIDATION_FAILED"
			row.ValidationErrors, _ = json.Marshal(validationErrors)
		} else {
			row.Status = "SUCCEEDED"
		}
		_ = s.repo.CreateAttempt(ctx, row)
		_ = s.repo.CreateUsage(ctx, &agentdomain.ModelUsageRecord{ID: uuid.NewString(), TenantID: tenantID, TaskID: taskID, PromptName: s.prompts.Name, PromptVersion: s.prompts.Version, SchemaVersion: s.prompts.SchemaVersion, Provider: s.client.Name(), Model: response.Model, InputTokens: response.Usage.InputTokens, OutputTokens: response.Usage.OutputTokens, EstimatedCost: decimal.Zero, LatencyMS: response.Latency.Milliseconds(), Status: usageStatus})
		if result != nil {
			return result, raw, nil
		}
	}
	return nil, raw, nil
}

func (s *Service) List(ctx context.Context, tenantID string) ([]agentdomain.AgentTask, error) {
	return s.repo.ListTasks(ctx, tenantID, 50)
}

func (s *Service) Get(ctx context.Context, tenantID, taskID string) (*repository.TaskDetails, error) {
	return s.repo.GetTask(ctx, tenantID, taskID)
}
func (s *Service) Health(_ context.Context) (*agentdomain.HealthStatus, error) {
	return &agentdomain.HealthStatus{Status: "UP", Provider: s.client.Name(), Details: []string{"structured_json", "result_validation", "restricted_tools"}}, nil
}
func (s *Service) Capabilities(_ context.Context) []string {
	return append([]string(nil), s.spec.Tools...)
}
func (s *Service) Cancel(ctx context.Context, taskID string) error {
	task, err := s.repo.FindTaskSystemByID(ctx, taskID)
	if err != nil {
		return err
	}
	return s.repo.TransitionTask(ctx, task.TenantID, taskID, "PENDING", "CANCELLED", "CANCELLED", "", nil)
}

func (s *Service) MarkRetrying(ctx context.Context, payload taskqueue.BusinessAnalysisPayload, err error, retryCount int) error {
	return s.repo.MarkTaskRetrying(ctx, payload.TenantID, payload.TaskID, err.Error(), retryCount)
}

func (s *Service) MarkFailed(ctx context.Context, payload taskqueue.BusinessAnalysisPayload, err error, retryCount int) error {
	return s.repo.MarkTaskFailed(ctx, payload.TenantID, payload.TaskID, err.Error(), retryCount)
}

func (s *Service) GetReport(ctx context.Context, tenantID, taskID string) (*domain.AnalysisReport, error) {
	return s.repo.GetReport(ctx, tenantID, taskID)
}
