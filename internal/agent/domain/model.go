package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type AgentTask struct {
	ID              string          `gorm:"type:char(36);primaryKey" json:"task_id"`
	TenantID        string          `gorm:"type:char(36);not null;uniqueIndex:uidx_agent_task_idempotency" json:"tenant_id"`
	WorkflowID      string          `gorm:"type:char(36);index" json:"workflow_id,omitempty"`
	ParentTaskID    string          `gorm:"type:char(36);index" json:"parent_task_id,omitempty"`
	AgentName       string          `gorm:"size:80;not null;default:business-agent;index" json:"agent_name"`
	GameID          string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID      string          `gorm:"type:char(36);not null;index" json:"campaign_id"`
	TaskType        string          `gorm:"size:50;not null" json:"task_type"`
	SchemaName      string          `gorm:"size:80;not null" json:"schema_name"`
	SchemaVersion   string          `gorm:"size:40;not null" json:"schema_version"`
	Status          string          `gorm:"size:30;not null;index" json:"status"`
	Priority        int             `gorm:"not null;default:0" json:"priority"`
	IdempotencyKey  string          `gorm:"size:255;not null;uniqueIndex:uidx_agent_task_idempotency" json:"idempotency_key"`
	InputJSON       json.RawMessage `gorm:"type:json" json:"input_json"`
	OutputJSON      json.RawMessage `gorm:"type:json" json:"output_json,omitempty"`
	ErrorMessage    string          `gorm:"type:text" json:"error_message,omitempty"`
	CurrentStep     string          `gorm:"size:80;not null" json:"current_step"`
	Attempt         int             `gorm:"not null;default:0" json:"attempt"`
	MaxAttempts     int             `gorm:"not null;default:2" json:"max_attempts"`
	QueueRetryCount int             `gorm:"not null;default:0" json:"queue_retry_count"`
	ScheduledAt     time.Time       `json:"scheduled_at"`
	QueuedAt        *time.Time      `json:"queued_at,omitempty"`
	LastHeartbeatAt *time.Time      `json:"last_heartbeat_at,omitempty"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	FinishedAt      *time.Time      `json:"finished_at,omitempty"`
	CreatedBy       string          `gorm:"type:char(36);not null" json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type AgentTaskAttempt struct {
	ID               string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID         string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	TaskID           string          `gorm:"type:char(36);not null;index" json:"task_id"`
	AttemptNumber    int             `gorm:"not null" json:"attempt_number"`
	Provider         string          `gorm:"size:50;not null" json:"provider"`
	Model            string          `gorm:"size:120;not null" json:"model"`
	Status           string          `gorm:"size:30;not null" json:"status"`
	RawResponseJSON  json.RawMessage `gorm:"type:json" json:"raw_response_json,omitempty"`
	ValidationErrors json.RawMessage `gorm:"type:json" json:"validation_errors,omitempty"`
	ErrorMessage     string          `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt        time.Time       `json:"started_at"`
	FinishedAt       time.Time       `json:"finished_at"`
	CreatedAt        time.Time       `json:"created_at"`
}

type ModelUsageRecord struct {
	ID            string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID      string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	TaskID        string          `gorm:"type:char(36);not null;index" json:"task_id"`
	PromptName    string          `gorm:"size:80;not null" json:"prompt_name"`
	PromptVersion string          `gorm:"size:40;not null" json:"prompt_version"`
	SchemaVersion string          `gorm:"size:40;not null" json:"schema_version"`
	Provider      string          `gorm:"size:50;not null" json:"provider"`
	Model         string          `gorm:"size:120;not null" json:"model"`
	InputTokens   int             `gorm:"not null" json:"input_tokens"`
	OutputTokens  int             `gorm:"not null" json:"output_tokens"`
	EstimatedCost decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"estimated_cost"`
	LatencyMS     int64           `gorm:"not null" json:"latency_ms"`
	Status        string          `gorm:"size:30;not null" json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
}

type TaskOutbox struct {
	ID            string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID      string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	TaskID        string          `gorm:"type:char(36);not null;uniqueIndex" json:"task_id"`
	TaskType      string          `gorm:"size:80;not null" json:"task_type"`
	PayloadJSON   json.RawMessage `gorm:"type:json;not null" json:"payload_json"`
	Status        string          `gorm:"size:30;not null;index" json:"status"`
	DispatchCount int             `gorm:"not null;default:0" json:"dispatch_count"`
	NextAttemptAt time.Time       `gorm:"not null;index" json:"next_attempt_at"`
	LastError     string          `gorm:"type:text" json:"last_error,omitempty"`
	PublishedAt   *time.Time      `json:"published_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func (AgentTask) TableName() string        { return "agent_tasks" }
func (AgentTaskAttempt) TableName() string { return "agent_task_attempts" }
func (ModelUsageRecord) TableName() string { return "model_usage_records" }
func (TaskOutbox) TableName() string       { return "task_outbox" }

type AgentSpec struct {
	Name          string        `json:"name"`
	Version       string        `json:"version"`
	Description   string        `json:"description"`
	ExecutionMode string        `json:"execution_mode"`
	Model         string        `json:"model,omitempty"`
	SystemPrompt  string        `json:"-"`
	Tools         []string      `json:"tools"`
	Permissions   []string      `json:"permissions"`
	MaxSteps      int           `json:"max_steps"`
	Timeout       time.Duration `json:"timeout"`
	InputSchema   string        `json:"input_schema,omitempty"`
	OutputSchema  string        `json:"output_schema,omitempty"`
}

type AgentInput struct {
	WorkflowID string          `json:"workflow_id"`
	TaskID     string          `json:"task_id"`
	TenantID   string          `json:"tenant_id"`
	UserID     string          `json:"user_id"`
	TraceID    string          `json:"trace_id"`
	Payload    json.RawMessage `json:"payload"`
}

type AgentResult struct {
	TaskID         string          `json:"task_id"`
	ExternalTaskID string          `json:"external_task_id,omitempty"`
	AgentName      string          `json:"agent_name"`
	Status         string          `json:"status"`
	Output         json.RawMessage `json:"output,omitempty"`
}

type HealthStatus struct {
	Status   string   `json:"status"`
	Provider string   `json:"provider"`
	Details  []string `json:"details"`
}

type AgentExecutor interface {
	Execute(ctx context.Context, input AgentInput) (*AgentResult, error)
	Health(ctx context.Context) (*HealthStatus, error)
	Capabilities(ctx context.Context) []string
}

type CancellableAgent interface {
	Cancel(ctx context.Context, taskID string) error
}
