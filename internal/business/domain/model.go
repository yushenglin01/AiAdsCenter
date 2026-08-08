package domain

import (
	"encoding/json"
	"time"
)

type BusinessFinding struct {
	ID           string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID     string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	TaskID       string          `gorm:"type:char(36);not null;index" json:"task_id"`
	GameID       string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID   string          `gorm:"type:char(36);not null;index" json:"campaign_id"`
	Type         string          `gorm:"size:60;not null" json:"type"`
	RuleCode     string          `gorm:"size:80;not null" json:"rule_code"`
	Severity     string          `gorm:"size:20;not null" json:"severity"`
	Conclusion   string          `gorm:"size:240;not null" json:"conclusion"`
	Description  string          `gorm:"type:text;not null" json:"description"`
	EvidenceJSON json.RawMessage `gorm:"type:json" json:"evidence"`
	Confidence   float64         `gorm:"type:decimal(5,4);not null" json:"confidence"`
	CreatedAt    time.Time       `json:"created_at"`
}

type Recommendation struct {
	ID               string    `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID         string    `gorm:"type:char(36);not null;index" json:"tenant_id"`
	TaskID           string    `gorm:"type:char(36);not null;index" json:"task_id"`
	CampaignID       string    `gorm:"type:char(36);not null;index" json:"campaign_id"`
	Action           string    `gorm:"size:50;not null" json:"action"`
	Description      string    `gorm:"type:text;not null" json:"description"`
	Priority         int       `gorm:"not null" json:"priority"`
	RiskLevel        string    `gorm:"size:20;not null" json:"risk_level"`
	RequiresApproval bool      `gorm:"not null" json:"requires_approval"`
	SuggestedValue   string    `gorm:"size:80" json:"suggested_value,omitempty"`
	Status           string    `gorm:"size:30;not null" json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

type ApprovalRequest struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID         string     `gorm:"type:char(36);not null;index" json:"tenant_id"`
	RecommendationID string     `gorm:"type:char(36);not null;uniqueIndex" json:"recommendation_id"`
	TaskID           string     `gorm:"type:char(36);not null;index" json:"task_id"`
	ReportID         string     `gorm:"type:char(36);index" json:"report_id,omitempty"`
	Action           string     `gorm:"size:50;not null" json:"action"`
	SuggestedValue   string     `gorm:"size:80" json:"suggested_value,omitempty"`
	Reason           string     `gorm:"type:text" json:"reason,omitempty"`
	RiskLevel        string     `gorm:"size:20;not null" json:"risk_level"`
	Status           string     `gorm:"size:30;not null" json:"status"`
	RequestedBy      string     `gorm:"type:char(36);not null" json:"requested_by"`
	DecidedBy        string     `gorm:"column:reviewed_by;type:char(36)" json:"decided_by,omitempty"`
	DecisionComment  string     `gorm:"column:review_comment;type:text" json:"decision_comment,omitempty"`
	DecisionVersion  int        `gorm:"not null" json:"decision_version"`
	DecidedAt        *time.Time `gorm:"column:reviewed_at" json:"decided_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type AnalysisReport struct {
	ID               string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID         string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	TaskID           string          `gorm:"type:char(36);not null;uniqueIndex" json:"task_id"`
	Title            string          `gorm:"size:240;not null" json:"title"`
	Summary          string          `gorm:"type:text;not null" json:"summary"`
	ContentMarkdown  string          `gorm:"type:longtext;not null" json:"content_markdown"`
	Status           string          `gorm:"size:30;not null" json:"status"`
	GeneratorAgent   string          `gorm:"size:80" json:"generator_agent"`
	GeneratorVersion string          `gorm:"size:40" json:"generator_version"`
	SourceDigest     string          `gorm:"size:64" json:"source_digest,omitempty"`
	ProvenanceJSON   json.RawMessage `gorm:"type:json" json:"provenance,omitempty"`
	GeneratedAt      *time.Time      `json:"generated_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func (BusinessFinding) TableName() string { return "business_findings" }
func (Recommendation) TableName() string  { return "recommendations" }
func (ApprovalRequest) TableName() string { return "approval_requests" }
func (AnalysisReport) TableName() string  { return "analysis_reports" }

type Evidence struct {
	Metric string `json:"metric"`
	Actual string `json:"actual"`
	Target string `json:"target,omitempty"`
}

type Finding struct {
	Type           string     `json:"type"`
	RuleCode       string     `json:"rule_code"`
	Severity       string     `json:"severity"`
	Conclusion     string     `json:"conclusion"`
	Description    string     `json:"description"`
	Evidence       []Evidence `json:"evidence"`
	PossibleCauses []string   `json:"possible_causes"`
	Impact         string     `json:"impact"`
	Confidence     float64    `json:"confidence"`
}

type RecommendationOutput struct {
	Action           string `json:"action"`
	Description      string `json:"description"`
	Priority         int    `json:"priority"`
	RiskLevel        string `json:"risk_level"`
	RequiresApproval bool   `json:"requires_approval"`
	SuggestedValue   string `json:"suggested_value,omitempty"`
}

type Result struct {
	Status          string                 `json:"status"`
	Summary         string                 `json:"summary"`
	Findings        []Finding              `json:"findings"`
	Recommendations []RecommendationOutput `json:"recommendations"`
}
