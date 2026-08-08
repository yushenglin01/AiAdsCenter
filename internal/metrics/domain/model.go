package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type CampaignDailyMetric struct {
	ID                string          `gorm:"type:char(36);primaryKey" json:"id"`
	TenantID          string          `gorm:"type:char(36);not null;uniqueIndex:uidx_campaign_daily_metric" json:"tenant_id"`
	GameID            string          `gorm:"type:char(36);not null;index" json:"game_id"`
	CampaignID        string          `gorm:"type:char(36);not null;uniqueIndex:uidx_campaign_daily_metric" json:"campaign_id"`
	Date              time.Time       `gorm:"type:date;not null;uniqueIndex:uidx_campaign_daily_metric" json:"date"`
	Country           string          `gorm:"size:2;not null;uniqueIndex:uidx_campaign_daily_metric" json:"country"`
	Currency          string          `gorm:"size:3;not null" json:"currency"`
	Spend             decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"spend"`
	DailyBudget       decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"daily_budget"`
	Impressions       int64           `gorm:"not null" json:"impressions"`
	Clicks            int64           `gorm:"not null" json:"clicks"`
	Installs          int64           `gorm:"not null" json:"installs"`
	Registrations     int64           `gorm:"not null" json:"registrations"`
	ActiveUsers       int64           `gorm:"not null" json:"active_users"`
	Payers            int64           `gorm:"not null" json:"payers"`
	RevenueD1         decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"revenue_d1"`
	RevenueD3         decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"revenue_d3"`
	RevenueD7         decimal.Decimal `gorm:"type:decimal(20,6);not null" json:"revenue_d7"`
	CTR               decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"ctr"`
	CVR               decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"cvr"`
	CPI               decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"cpi"`
	CPA               decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"cpa"`
	PayerRate         decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"payer_rate"`
	ROASD1            decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"roas_d1"`
	ROASD3            decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"roas_d3"`
	ROASD7            decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"roas_d7"`
	BudgetConsumption decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"budget_consumption_rate"`
	LTVD7             decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"ltv_d7"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (CampaignDailyMetric) TableName() string { return "campaign_daily_metrics" }

type MetricInput struct {
	Spend, DailyBudget, RevenueD1, RevenueD3, RevenueD7               decimal.Decimal
	Impressions, Clicks, Installs, Registrations, ActiveUsers, Payers int64
}

type MetricValues struct {
	CTR, CVR, CPI, CPA, PayerRate, ROASD1, ROASD3, ROASD7, BudgetConsumption, LTVD7 decimal.Decimal
}
