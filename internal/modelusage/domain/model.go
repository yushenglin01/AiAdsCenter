package domain

import "github.com/shopspring/decimal"

type ProviderSummary struct {
	Provider       string          `json:"provider"`
	Calls          int64           `json:"calls"`
	InputTokens    int64           `json:"input_tokens"`
	OutputTokens   int64           `json:"output_tokens"`
	EstimatedCost  decimal.Decimal `json:"estimated_cost"`
	AverageLatency float64         `json:"average_latency_ms"`
}

type Summary struct {
	Calls           int64             `json:"calls"`
	SuccessfulCalls int64             `json:"successful_calls"`
	InputTokens     int64             `json:"input_tokens"`
	OutputTokens    int64             `json:"output_tokens"`
	EstimatedCost   decimal.Decimal   `json:"estimated_cost"`
	AverageLatency  float64           `json:"average_latency_ms"`
	SuccessRate     float64           `json:"success_rate"`
	Providers       []ProviderSummary `json:"providers"`
}
