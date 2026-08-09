package domain

import "time"

const (
	ProviderAppsFlyer  = "APPSFLYER"
	ProviderAdjust     = "ADJUST"
	ConnectionActive   = "ACTIVE"
	ConnectionDisabled = "DISABLED"
	SyncProcessing     = "PROCESSING"
	SyncSucceeded      = "SUCCEEDED"
	SyncFailed         = "FAILED"
)

type Connection struct {
	ID            string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID      string     `gorm:"type:char(36);not null;uniqueIndex:uidx_mmp_connection" json:"tenant_id"`
	GameID        string     `gorm:"type:char(36);not null;uniqueIndex:uidx_mmp_connection" json:"game_id"`
	Provider      string     `gorm:"size:40;not null;uniqueIndex:uidx_mmp_connection" json:"provider"`
	ExternalAppID string     `gorm:"size:255;not null" json:"external_app_id"`
	Status        string     `gorm:"size:20;not null" json:"status"`
	CreatedBy     string     `gorm:"type:char(36);not null" json:"created_by"`
	UpdatedBy     string     `gorm:"type:char(36);not null" json:"updated_by"`
	LastSyncAt    *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (Connection) TableName() string { return "mmp_connections" }

type SyncRun struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID       string     `gorm:"type:char(36);not null;uniqueIndex:uidx_mmp_sync_key" json:"tenant_id"`
	ConnectionID   string     `gorm:"type:char(36);not null;index" json:"connection_id"`
	Provider       string     `gorm:"size:40;not null" json:"provider"`
	PeriodStart    time.Time  `gorm:"type:date;not null" json:"period_start"`
	PeriodEnd      time.Time  `gorm:"type:date;not null" json:"period_end"`
	Status         string     `gorm:"size:20;not null" json:"status"`
	IdempotencyKey string     `gorm:"size:255;not null;uniqueIndex:uidx_mmp_sync_key" json:"idempotency_key"`
	ImportJobID    string     `gorm:"type:char(36);index" json:"import_job_id,omitempty"`
	SourceRows     int        `gorm:"not null" json:"source_rows"`
	NormalizedRows int        `gorm:"not null" json:"normalized_rows"`
	SkippedRows    int        `gorm:"not null" json:"skipped_rows"`
	WarningMessage string     `gorm:"type:text" json:"warning_message,omitempty"`
	ErrorCode      string     `gorm:"size:50" json:"error_code,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	RequestedBy    string     `gorm:"type:char(36);not null" json:"requested_by"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (SyncRun) TableName() string { return "mmp_sync_runs" }

type ConnectionView struct {
	Connection
	CredentialConfigured bool   `json:"credential_configured"`
	Health               string `json:"health"`
}
