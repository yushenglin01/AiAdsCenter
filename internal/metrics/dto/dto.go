package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

type CampaignMetric struct {
	CampaignID        string          `json:"campaign_id"`
	CampaignName      string          `json:"campaign_name"`
	Country           string          `json:"country"`
	Currency          string          `json:"currency"`
	Spend             decimal.Decimal `json:"spend"`
	Impressions       int64           `json:"impressions"`
	Clicks            int64           `json:"clicks"`
	Installs          int64           `json:"installs"`
	Registrations     int64           `json:"registrations"`
	ActiveUsers       int64           `json:"active_users"`
	Payers            int64           `json:"payers"`
	RevenueD1         decimal.Decimal `json:"revenue_d1"`
	RevenueD3         decimal.Decimal `json:"revenue_d3"`
	RevenueD7         decimal.Decimal `json:"revenue_d7"`
	DailyBudget       decimal.Decimal `json:"daily_budget"`
	CTR               decimal.Decimal `json:"ctr"`
	CVR               decimal.Decimal `json:"cvr"`
	CPI               decimal.Decimal `json:"cpi"`
	CPA               decimal.Decimal `json:"cpa"`
	PayerRate         decimal.Decimal `json:"payer_rate"`
	ROASD1            decimal.Decimal `json:"roas_d1"`
	ROASD3            decimal.Decimal `json:"roas_d3"`
	ROASD7            decimal.Decimal `json:"roas_d7"`
	BudgetConsumption decimal.Decimal `json:"budget_consumption_rate"`
	LTVD7             decimal.Decimal `json:"ltv_d7"`
}

type TrendPoint struct {
	Date      time.Time       `json:"date"`
	Spend     decimal.Decimal `json:"spend"`
	RevenueD1 decimal.Decimal `json:"revenue_d1"`
	RevenueD7 decimal.Decimal `json:"revenue_d7"`
	CPI       decimal.Decimal `json:"cpi"`
	ROASD1    decimal.Decimal `json:"roas_d1"`
	ROASD7    decimal.Decimal `json:"roas_d7"`
}
type Overview struct {
	CampaignMetric
	HighRiskCampaigns    int64 `json:"high_risk_campaigns"`
	AttributionAnomalies int64 `json:"attribution_anomalies"`
	FatiguedCreatives    int64 `json:"fatigued_creatives"`
}
