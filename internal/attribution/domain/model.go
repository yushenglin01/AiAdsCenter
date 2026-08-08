package domain

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type Finding struct {
	ID             string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID       string          `gorm:"type:char(36);not null;index" json:"tenant_id"`
	GameID         string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID     string          `gorm:"type:char(36);not null;index" json:"campaign_id"`
	CampaignName   string          `gorm:"size:160;not null" json:"campaign_name"`
	RuleCode       string          `gorm:"size:80;not null" json:"rule_code"`
	Severity       string          `gorm:"size:20;not null" json:"severity"`
	Title          string          `gorm:"size:200;not null" json:"title"`
	Description    string          `gorm:"type:text;not null" json:"description"`
	DifferenceRate decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"difference_rate"`
	Evidence       json.RawMessage `gorm:"type:json" json:"evidence"`
	CreatedAt      time.Time       `json:"created_at"`
}

func (Finding) TableName() string { return "attribution_findings" }
