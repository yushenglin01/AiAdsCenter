package domain

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type BusinessBenchmark struct {
	ID          string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string          `gorm:"type:char(36);not null;uniqueIndex:uidx_benchmark" json:"tenant_id"`
	GameID      string          `gorm:"type:char(36);not null;uniqueIndex:uidx_benchmark" json:"game_id"`
	CampaignID  string          `gorm:"type:char(36);not null;default:'';uniqueIndex:uidx_benchmark" json:"campaign_id"`
	MetricCode  string          `gorm:"size:80;not null;uniqueIndex:uidx_benchmark" json:"metric_code"`
	Value       decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"value"`
	PeriodStart time.Time       `gorm:"type:date" json:"period_start"`
	PeriodEnd   time.Time       `gorm:"type:date" json:"period_end"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
type AnalysisRule struct {
	ID              string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID        string          `gorm:"type:char(36);not null;uniqueIndex:uidx_rule_code" json:"tenant_id"`
	Code            string          `gorm:"size:80;not null;uniqueIndex:uidx_rule_code" json:"code"`
	Category        string          `gorm:"size:40;not null" json:"category"`
	Name            string          `gorm:"size:160;not null" json:"name"`
	Severity        string          `gorm:"size:20;not null" json:"severity"`
	Threshold       decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"threshold"`
	ConsecutiveDays int             `gorm:"not null;default:1" json:"consecutive_days"`
	Enabled         bool            `gorm:"not null;default:true" json:"enabled"`
	ConfigJSON      json.RawMessage `gorm:"type:json" json:"config"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       gorm.DeletedAt  `gorm:"index" json:"-"`
}
type RuleFinding struct {
	ID          string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	GameID      string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID  string          `gorm:"type:char(36);not null;index" json:"campaign_id"`
	RuleCode    string          `gorm:"size:80;not null" json:"rule_code"`
	Severity    string          `gorm:"size:20;not null" json:"severity"`
	Title       string          `gorm:"size:200;not null" json:"title"`
	Description string          `gorm:"type:text;not null" json:"description"`
	Evidence    json.RawMessage `gorm:"type:json" json:"evidence"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (BusinessBenchmark) TableName() string { return "business_benchmarks" }
func (AnalysisRule) TableName() string      { return "analysis_rules" }
func (RuleFinding) TableName() string       { return "rule_findings" }
