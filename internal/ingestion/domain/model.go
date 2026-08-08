package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type ImportJob struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID       string     `gorm:"type:char(36);not null;index" json:"tenant_id"`
	GameID         string     `gorm:"type:char(36);not null;index" json:"game_id"`
	ImportType     string     `gorm:"size:40;not null" json:"import_type"`
	Source         string     `gorm:"size:40;not null" json:"source"`
	FileName       string     `gorm:"size:255;not null" json:"file_name"`
	FileHash       string     `gorm:"size:64;not null" json:"file_hash"`
	IdempotencyKey string     `gorm:"size:255;not null;uniqueIndex" json:"idempotency_key"`
	BatchID        string     `gorm:"size:128;index" json:"batch_id,omitempty"`
	ProducerSystem string     `gorm:"size:80;index" json:"producer_system,omitempty"`
	SchemaVersion  string     `gorm:"size:20" json:"schema_version,omitempty"`
	PeriodStart    time.Time  `gorm:"type:date;not null" json:"period_start"`
	PeriodEnd      *time.Time `gorm:"type:date" json:"period_end,omitempty"`
	CollectedAt    *time.Time `json:"collected_at,omitempty"`
	Status         string     `gorm:"size:20;not null" json:"status"`
	TotalRows      int        `gorm:"not null" json:"total_rows"`
	ImportedRows   int        `gorm:"not null" json:"imported_rows"`
	SkippedRows    int        `gorm:"not null" json:"skipped_rows"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedBy      string     `gorm:"type:char(36);not null" json:"created_by"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type IngestionMessage struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID       string     `gorm:"type:char(36);not null;uniqueIndex:uidx_ingestion_event" json:"tenant_id"`
	EventID        string     `gorm:"size:128;not null;uniqueIndex:uidx_ingestion_event" json:"event_id"`
	ProducerSystem string     `gorm:"size:80;not null;uniqueIndex:uidx_ingestion_event" json:"producer_system"`
	SchemaVersion  string     `gorm:"size:20;not null" json:"schema_version"`
	Topic          string     `gorm:"size:200;not null;uniqueIndex:uidx_kafka_position" json:"topic"`
	Partition      int32      `gorm:"column:partition_number;not null;uniqueIndex:uidx_kafka_position" json:"partition"`
	Offset         int64      `gorm:"column:kafka_offset;not null;uniqueIndex:uidx_kafka_position" json:"offset"`
	PayloadHash    string     `gorm:"size:64;not null" json:"payload_hash"`
	Status         string     `gorm:"size:30;not null;index" json:"status"`
	AttemptCount   int        `gorm:"not null" json:"attempt_count"`
	ErrorCode      string     `gorm:"size:80" json:"error_code,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type AnalysisWindow struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID         string     `gorm:"type:char(36);not null;uniqueIndex:uidx_analysis_window" json:"tenant_id"`
	GameID           string     `gorm:"type:char(36);not null;uniqueIndex:uidx_analysis_window" json:"game_id"`
	PeriodStart      time.Time  `gorm:"type:date;not null;uniqueIndex:uidx_analysis_window" json:"period_start"`
	PeriodEnd        time.Time  `gorm:"type:date;not null;uniqueIndex:uidx_analysis_window" json:"period_end"`
	ReceivedDatasets []byte     `gorm:"type:json;not null" json:"received_datasets"`
	Status           string     `gorm:"size:30;not null;index" json:"status"`
	Version          int        `gorm:"not null" json:"version"`
	AttemptCount     int        `gorm:"not null" json:"attempt_count"`
	NextRunAt        time.Time  `gorm:"not null;index" json:"next_run_at"`
	LockedUntil      *time.Time `json:"locked_until,omitempty"`
	ClaimToken       string     `gorm:"type:char(36)" json:"-"`
	LastError        string     `gorm:"type:text" json:"last_error,omitempty"`
	ProcessedAt      *time.Time `json:"processed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type RawAdMetric struct {
	ID          string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string          `gorm:"type:char(36);not null;uniqueIndex:uidx_raw_ad_row" json:"tenant_id"`
	ImportJobID string          `gorm:"type:char(36);not null;index" json:"import_job_id"`
	Source      string          `gorm:"size:40;not null;uniqueIndex:uidx_raw_ad_row" json:"source"`
	GameID      string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID  string          `gorm:"type:char(36);not null;index;uniqueIndex:uidx_raw_ad_row" json:"campaign_id"`
	Date        time.Time       `gorm:"type:date;not null;uniqueIndex:uidx_raw_ad_row" json:"date"`
	Country     string          `gorm:"size:2;not null;uniqueIndex:uidx_raw_ad_row" json:"country"`
	Currency    string          `gorm:"size:3;not null" json:"currency"`
	Spend       decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"spend"`
	Impressions int64           `gorm:"not null" json:"impressions"`
	Clicks      int64           `gorm:"not null" json:"clicks"`
	Installs    int64           `gorm:"not null" json:"installs"`
	CreatedAt   time.Time       `json:"created_at"`
}

type NormalizedAdMetric RawAdMetric

func (NormalizedAdMetric) TableName() string { return "normalized_ad_metrics" }

type MMPMetric struct {
	ID          string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string          `gorm:"type:char(36);not null;uniqueIndex:uidx_mmp_row" json:"tenant_id"`
	ImportJobID string          `gorm:"type:char(36);not null;index" json:"import_job_id"`
	Source      string          `gorm:"size:40;not null;uniqueIndex:uidx_mmp_row" json:"source"`
	GameID      string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID  string          `gorm:"type:char(36);not null;uniqueIndex:uidx_mmp_row" json:"campaign_id"`
	Date        time.Time       `gorm:"type:date;not null;uniqueIndex:uidx_mmp_row" json:"date"`
	Country     string          `gorm:"size:2;not null;uniqueIndex:uidx_mmp_row" json:"country"`
	Installs    int64           `gorm:"not null" json:"installs"`
	Activations int64           `gorm:"not null" json:"activations"`
	Payers      int64           `gorm:"not null" json:"payers"`
	Revenue     decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"revenue"`
	Currency    string          `gorm:"size:3;not null" json:"currency"`
	CreatedAt   time.Time       `json:"created_at"`
}

type GameRevenueMetric struct {
	ID            string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID      string          `gorm:"type:char(36);not null;uniqueIndex:uidx_revenue_row" json:"tenant_id"`
	ImportJobID   string          `gorm:"type:char(36);not null;index" json:"import_job_id"`
	GameID        string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID    string          `gorm:"type:char(36);not null;uniqueIndex:uidx_revenue_row" json:"campaign_id"`
	Date          time.Time       `gorm:"type:date;not null;uniqueIndex:uidx_revenue_row" json:"date"`
	Country       string          `gorm:"size:2;not null;uniqueIndex:uidx_revenue_row" json:"country"`
	Registrations int64           `gorm:"not null" json:"registrations"`
	ActiveUsers   int64           `gorm:"not null" json:"active_users"`
	Payers        int64           `gorm:"not null" json:"payers"`
	RevenueD1     decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"revenue_d1"`
	RevenueD3     decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"revenue_d3"`
	RevenueD7     decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"revenue_d7"`
	Currency      string          `gorm:"size:3;not null" json:"currency"`
	CreatedAt     time.Time       `json:"created_at"`
}

type CreativeDailyMetric struct {
	ID          string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string          `gorm:"type:char(36);not null;uniqueIndex:uidx_creative_metric_row" json:"tenant_id"`
	ImportJobID string          `gorm:"type:char(36);not null;index" json:"import_job_id"`
	GameID      string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID  string          `gorm:"type:char(36);not null;index" json:"campaign_id"`
	CreativeID  string          `gorm:"type:char(36);not null;uniqueIndex:uidx_creative_metric_row" json:"creative_id"`
	Date        time.Time       `gorm:"type:date;not null;uniqueIndex:uidx_creative_metric_row" json:"date"`
	Spend       decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"spend"`
	Impressions int64           `gorm:"not null" json:"impressions"`
	Clicks      int64           `gorm:"not null" json:"clicks"`
	Installs    int64           `gorm:"not null" json:"installs"`
	Frequency   decimal.Decimal `gorm:"type:decimal(12,6);not null" json:"frequency"`
	Currency    string          `gorm:"size:3;not null" json:"currency"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (ImportJob) TableName() string           { return "data_import_jobs" }
func (IngestionMessage) TableName() string    { return "ingestion_messages" }
func (AnalysisWindow) TableName() string      { return "analysis_windows" }
func (RawAdMetric) TableName() string         { return "raw_ad_metrics" }
func (MMPMetric) TableName() string           { return "mmp_metrics" }
func (GameRevenueMetric) TableName() string   { return "game_revenue_metrics" }
func (CreativeDailyMetric) TableName() string { return "creative_daily_metrics" }
