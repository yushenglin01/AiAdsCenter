package domain

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Channel struct {
	ID        string         `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID  string         `gorm:"type:char(36);not null;uniqueIndex:uidx_channel_tenant_code" json:"tenant_id"`
	Code      string         `gorm:"size:40;not null;uniqueIndex:uidx_channel_tenant_code" json:"code"`
	Name      string         `gorm:"size:80;not null" json:"name"`
	Provider  string         `gorm:"size:40;not null" json:"provider"`
	Status    string         `gorm:"size:20;not null" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Campaign struct {
	ID          string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID    string          `gorm:"type:char(36);not null;uniqueIndex:uidx_campaign_tenant_external" json:"tenant_id"`
	GameID      string          `gorm:"type:char(36);not null;index" json:"game_id"`
	ChannelID   string          `gorm:"type:char(36);not null;index" json:"channel_id"`
	ExternalID  string          `gorm:"size:120;not null;uniqueIndex:uidx_campaign_tenant_external" json:"external_id"`
	Name        string          `gorm:"size:160;not null" json:"name"`
	Country     string          `gorm:"size:2;not null" json:"country"`
	DailyBudget decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"daily_budget"`
	Currency    string          `gorm:"size:3;not null" json:"currency"`
	Status      string          `gorm:"size:20;not null" json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (Channel) TableName() string  { return "channels" }
func (Campaign) TableName() string { return "campaigns" }
