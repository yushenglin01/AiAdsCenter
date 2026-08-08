package domain

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Creative struct {
	ID         string         `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID   string         `gorm:"type:char(36);not null;uniqueIndex:uidx_creative_tenant_external" json:"tenant_id"`
	CampaignID string         `gorm:"type:char(36);not null;index" json:"campaign_id"`
	ExternalID string         `gorm:"size:120;not null;uniqueIndex:uidx_creative_tenant_external" json:"external_id"`
	Name       string         `gorm:"size:160;not null" json:"name"`
	Type       string         `gorm:"size:30;not null" json:"type"`
	Status     string         `gorm:"size:20;not null" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Creative) TableName() string { return "creatives" }

type Finding struct {
	ID           string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID     string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	GameID       string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID   string          `gorm:"type:char(36);not null;index" json:"campaign_id"`
	CreativeID   string          `gorm:"type:char(36);not null;index" json:"creative_id"`
	CreativeName string          `gorm:"size:160;not null" json:"creative_name"`
	RuleCode     string          `gorm:"size:80;not null" json:"rule_code"`
	Severity     string          `gorm:"size:20;not null" json:"severity"`
	Title        string          `gorm:"size:200;not null" json:"title"`
	Description  string          `gorm:"type:text;not null" json:"description"`
	FatigueScore decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"fatigue_score"`
	CTRChange7D  decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"ctr_change_7d"`
	Frequency    decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"frequency"`
	Evidence     json.RawMessage `gorm:"type:json" json:"evidence"`
	CreatedAt    time.Time       `json:"created_at"`
}

func (Finding) TableName() string { return "creative_findings" }
