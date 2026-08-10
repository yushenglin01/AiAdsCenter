package domain

import "time"

const (
	CategoryPolicy     = "POLICY"
	CategoryCompetitor = "COMPETITOR"
	CategoryMarket     = "MARKET"

	StatusPending  = "PENDING"
	StatusVerified = "VERIFIED"
	StatusRejected = "REJECTED"

	ScheduleRunProcessing = "PROCESSING"
	ScheduleRunSucceeded  = "SUCCEEDED"
	ScheduleRunFailed     = "FAILED"
)

type Source struct {
	ID                  string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID            string     `gorm:"type:char(36);not null;uniqueIndex:uidx_research_source_hash" json:"tenant_id"`
	GameID              string     `gorm:"type:char(36);index" json:"game_id,omitempty"`
	CampaignID          string     `gorm:"type:char(36);index" json:"campaign_id,omitempty"`
	Category            string     `gorm:"size:30;not null;index" json:"category"`
	Title               string     `gorm:"size:300;not null" json:"title"`
	Summary             string     `gorm:"type:text;not null" json:"summary"`
	SourceURL           string     `gorm:"size:1000;not null" json:"source_url"`
	DiscoveryMethod     string     `gorm:"size:30;not null;default:MANUAL" json:"discovery_method"`
	DiscoveryProvider   string     `gorm:"size:50" json:"discovery_provider,omitempty"`
	DiscoveryQueryHash  string     `gorm:"type:char(64)" json:"discovery_query_hash,omitempty"`
	DiscoveryScheduleID string     `gorm:"type:char(36);index" json:"discovery_schedule_id,omitempty"`
	DiscoveredAt        *time.Time `json:"discovered_at,omitempty"`
	Publisher           string     `gorm:"size:200;not null" json:"publisher"`
	PublishedAt         time.Time  `gorm:"not null;index" json:"published_at"`
	ContentHash         string     `gorm:"size:64;not null;uniqueIndex:uidx_research_source_hash" json:"content_hash"`
	Status              string     `gorm:"size:20;not null;index" json:"status"`
	ReviewComment       string     `gorm:"type:text" json:"review_comment,omitempty"`
	CreatedBy           string     `gorm:"type:char(36);not null" json:"created_by"`
	ReviewedBy          string     `gorm:"type:char(36)" json:"reviewed_by,omitempty"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (Source) TableName() string { return "research_sources" }

type Evidence struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	SourceURL   string    `json:"source_url"`
	Publisher   string    `json:"publisher"`
	PublishedAt time.Time `json:"published_at"`
	VerifiedAt  time.Time `json:"verified_at"`
}

type Filter struct {
	GameID     string
	CampaignID string
	Category   string
	Status     string
	Limit      int
}

type Schedule struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID         string     `gorm:"type:char(36);not null;index" json:"tenant_id"`
	Name             string     `gorm:"size:120;not null" json:"name"`
	GameID           string     `gorm:"type:char(36);index" json:"game_id,omitempty"`
	CampaignID       string     `gorm:"type:char(36);index" json:"campaign_id,omitempty"`
	Category         string     `gorm:"size:30;not null" json:"category"`
	Query            string     `gorm:"size:400;not null" json:"query"`
	QueryHash        string     `gorm:"type:char(64);not null" json:"query_hash"`
	IdempotencyKey   string     `gorm:"type:char(64);not null" json:"-"`
	Country          string     `gorm:"size:2" json:"country,omitempty"`
	SearchLang       string     `gorm:"size:12" json:"search_lang,omitempty"`
	Freshness        string     `gorm:"size:2" json:"freshness,omitempty"`
	ResultCount      int        `gorm:"not null" json:"result_count"`
	IntervalMinutes  int        `gorm:"not null" json:"interval_minutes"`
	Enabled          bool       `gorm:"not null" json:"enabled"`
	NextRunAt        *time.Time `json:"next_run_at,omitempty"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	LastStatus       string     `gorm:"size:20" json:"last_status,omitempty"`
	LastErrorCode    string     `gorm:"size:50" json:"last_error_code,omitempty"`
	LastErrorMessage string     `gorm:"type:text" json:"last_error_message,omitempty"`
	LockToken        string     `gorm:"type:char(36)" json:"-"`
	LockedUntil      *time.Time `json:"-"`
	CreatedBy        string     `gorm:"type:char(36);not null" json:"created_by"`
	UpdatedBy        string     `gorm:"type:char(36);not null" json:"updated_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (Schedule) TableName() string { return "research_schedules" }

type ScheduleRun struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID       string     `gorm:"type:char(36);not null;index" json:"tenant_id"`
	ScheduleID     string     `gorm:"type:char(36);not null;index" json:"schedule_id"`
	ScheduledFor   time.Time  `gorm:"not null" json:"scheduled_for"`
	Status         string     `gorm:"size:20;not null" json:"status"`
	Provider       string     `gorm:"size:50;not null" json:"provider"`
	QueryHash      string     `gorm:"type:char(64);not null" json:"query_hash"`
	ResultCount    int        `gorm:"not null" json:"result_count"`
	ImportedCount  int        `gorm:"not null" json:"imported_count"`
	DuplicateCount int        `gorm:"not null" json:"duplicate_count"`
	SkippedCount   int        `gorm:"not null" json:"skipped_count"`
	ErrorCode      string     `gorm:"size:50" json:"error_code,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	RequestedBy    string     `gorm:"type:char(36);not null" json:"requested_by"`
	ClaimToken     string     `gorm:"type:char(36)" json:"-"`
	StartedAt      time.Time  `gorm:"not null" json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (ScheduleRun) TableName() string { return "research_schedule_runs" }
