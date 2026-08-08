package domain

import (
	"encoding/json"
	"time"
)

type Run struct {
	ID             string          `gorm:"type:char(36);primaryKey" json:"workflow_id"`
	TenantID       string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	WorkflowType   string          `gorm:"size:60;not null" json:"workflow_type"`
	GameID         string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID     string          `gorm:"type:char(36);not null;index" json:"campaign_id"`
	Status         string          `gorm:"size:30;not null;index" json:"status"`
	CurrentStep    string          `gorm:"size:80;not null" json:"current_step"`
	IdempotencyKey string          `gorm:"size:255;not null;uniqueIndex:uidx_workflow_run_idempotency" json:"idempotency_key"`
	BusinessTaskID string          `gorm:"type:char(36);index" json:"business_task_id,omitempty"`
	InputJSON      json.RawMessage `gorm:"type:json" json:"input_json,omitempty"`
	OutputJSON     json.RawMessage `gorm:"type:json" json:"output_json,omitempty"`
	ErrorMessage   string          `gorm:"type:text" json:"error_message,omitempty"`
	TriggeredBy    string          `gorm:"type:char(36);not null" json:"triggered_by"`
	TraceID        string          `gorm:"size:80" json:"trace_id,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (Run) TableName() string { return "workflow_runs" }

type Step struct {
	ID             string          `gorm:"type:char(36);primaryKey" json:"step_id"`
	TenantID       string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	WorkflowID     string          `gorm:"type:char(36);not null;index" json:"workflow_id"`
	AgentName      string          `gorm:"size:80;not null" json:"agent_name"`
	SequenceNumber int             `gorm:"not null" json:"sequence_number"`
	Status         string          `gorm:"size:30;not null" json:"status"`
	ExecutionMode  string          `gorm:"size:40;not null" json:"execution_mode"`
	ExternalTaskID string          `gorm:"type:char(36);index" json:"external_task_id,omitempty"`
	InputJSON      json.RawMessage `gorm:"type:json" json:"input_json,omitempty"`
	OutputJSON     json.RawMessage `gorm:"type:json" json:"output_json,omitempty"`
	ErrorMessage   string          `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (Step) TableName() string { return "workflow_steps" }

type Details struct {
	Run   Run    `json:"workflow"`
	Steps []Step `json:"steps"`
}
