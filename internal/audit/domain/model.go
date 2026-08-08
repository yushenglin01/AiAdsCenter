package domain

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID           string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID     string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	ActorID      string          `gorm:"type:char(36);not null;index" json:"actor_id"`
	ActorType    string          `gorm:"size:30;not null" json:"actor_type"`
	Action       string          `gorm:"size:80;not null;index" json:"action"`
	ResourceType string          `gorm:"size:60;not null;index" json:"resource_type"`
	ResourceID   string          `gorm:"type:char(36);not null;index" json:"resource_id"`
	TaskID       string          `gorm:"type:char(36);index" json:"task_id,omitempty"`
	BeforeJSON   json.RawMessage `gorm:"type:json" json:"before,omitempty"`
	AfterJSON    json.RawMessage `gorm:"type:json" json:"after,omitempty"`
	MetadataJSON json.RawMessage `gorm:"type:json" json:"metadata,omitempty"`
	RequestID    string          `gorm:"size:80" json:"request_id,omitempty"`
	TraceID      string          `gorm:"size:80" json:"trace_id,omitempty"`
	IPAddress    string          `gorm:"size:80" json:"ip_address,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

type RecordInput struct {
	TenantID     string
	ActorID      string
	ActorType    string
	Action       string
	ResourceType string
	ResourceID   string
	TaskID       string
	Before       any
	After        any
	Metadata     any
	RequestID    string
	TraceID      string
	IPAddress    string
}

type Filter struct {
	Action       string
	ResourceType string
	ActorID      string
	TaskID       string
	Limit        int
}
