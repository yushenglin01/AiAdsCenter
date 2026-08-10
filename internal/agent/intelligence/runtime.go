package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	"github.com/example/adnova/internal/llm"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrDisabled = errors.New("agent LLM enhancement is disabled")

type UsageRecorder interface {
	Create(context.Context, *agentdomain.ModelUsageRecord) error
	UpdateStatus(context.Context, string, string) error
}

type Runtime struct {
	client   llm.Client
	model    string
	enabled  map[string]bool
	recorder UsageRecorder
}

type Request struct {
	AgentName   string
	TenantID    string
	TaskID      string
	TraceID     string
	Contract    Contract
	UserPrompt  string
	InputJSON   json.RawMessage
	Temperature float64
	MaxTokens   int
}

type Result struct {
	Content       json.RawMessage `json:"-"`
	Provider      string          `json:"provider"`
	Model         string          `json:"model"`
	PromptVersion string          `json:"prompt_version"`
	SchemaVersion string          `json:"schema_version"`
	UsageRecordID string          `json:"-"`
}

func New(client llm.Client, model string, enabled map[string]bool, recorder UsageRecorder) *Runtime {
	flags := make(map[string]bool, len(enabled))
	for name, value := range enabled {
		flags[name] = value
	}
	return &Runtime{client: client, model: model, enabled: flags, recorder: recorder}
}

func (r *Runtime) Enabled(agentName string) bool {
	return r != nil && r.client != nil && r.client.Name() != "mock" && r.enabled[agentName]
}

func (r *Runtime) Provider() string {
	if r == nil || r.client == nil {
		return "none"
	}
	return r.client.Name()
}

func (r *Runtime) Model() string {
	if r == nil {
		return ""
	}
	return r.model
}

func (r *Runtime) Generate(ctx context.Context, request Request) (*Result, error) {
	if !r.Enabled(request.AgentName) {
		return nil, ErrDisabled
	}
	taskID := request.TaskID
	if taskID == "" {
		taskID = uuid.NewString()
	}
	started := time.Now()
	response, err := r.client.GenerateStructured(ctx, llm.GenerateRequest{
		Model: r.model, SystemPrompt: request.Contract.SystemPrompt, UserPrompt: request.UserPrompt,
		InputJSON: request.InputJSON, OutputSchema: request.Contract.OutputSchema,
		Temperature: request.Temperature, MaxTokens: request.MaxTokens, TraceID: request.TraceID,
	})
	if err != nil {
		r.record(ctx, request, taskID, r.model, llm.Usage{}, time.Since(started), "FAILED")
		return nil, err
	}
	usageRecordID := r.record(ctx, request, taskID, response.Model, response.Usage, response.Latency, "SUCCEEDED")
	return &Result{
		Content: response.Content, Provider: r.client.Name(), Model: response.Model,
		PromptVersion: request.Contract.PromptVersion, SchemaVersion: request.Contract.SchemaVersion,
		UsageRecordID: usageRecordID,
	}, nil
}

func (r *Runtime) MarkValidationFailed(ctx context.Context, result *Result) {
	if r == nil || r.recorder == nil || result == nil || result.UsageRecordID == "" {
		return
	}
	_ = r.recorder.UpdateStatus(ctx, result.UsageRecordID, "VALIDATION_FAILED")
}

func (r *Runtime) record(ctx context.Context, request Request, taskID, model string, usage llm.Usage, latency time.Duration, status string) string {
	if r.recorder == nil || request.TenantID == "" {
		return ""
	}
	id := uuid.NewString()
	_ = r.recorder.Create(ctx, &agentdomain.ModelUsageRecord{
		ID: id, TenantID: request.TenantID, TaskID: taskID,
		PromptName: request.Contract.PromptName, PromptVersion: request.Contract.PromptVersion,
		SchemaVersion: request.Contract.SchemaVersion, Provider: r.client.Name(), Model: model,
		InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens,
		EstimatedCost: decimal.Zero, LatencyMS: latency.Milliseconds(), Status: status,
	})
	return id
}

func FailureCategory(err error) string {
	if errors.Is(err, ErrDisabled) {
		return "DISABLED"
	}
	var clientErr *llm.ClientError
	if errors.As(err, &clientErr) {
		return string(clientErr.Category)
	}
	return "INTERNAL"
}
