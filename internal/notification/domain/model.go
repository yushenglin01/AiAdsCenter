package domain

import (
	"encoding/json"
	"time"
)

const (
	StatusUnread = "UNREAD"
	StatusRead   = "READ"
)

type Notification struct {
	ID           string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID     string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	WorkflowID   string          `gorm:"type:char(36);not null;index" json:"workflow_id"`
	EventType    string          `gorm:"size:80;not null" json:"event_type"`
	Channel      string          `gorm:"size:30;not null" json:"channel"`
	Title        string          `gorm:"size:200;not null" json:"title"`
	Message      string          `gorm:"type:text;not null" json:"message"`
	Status       string          `gorm:"size:20;not null;index" json:"status"`
	MetadataJSON json.RawMessage `gorm:"type:json" json:"metadata,omitempty"`
	ReadBy       string          `gorm:"type:char(36)" json:"read_by,omitempty"`
	ReadAt       *time.Time      `json:"read_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (Notification) TableName() string { return "workflow_notifications" }
