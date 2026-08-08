package service

import (
	"testing"
	"time"

	metricsdomain "github.com/example/adnova/internal/metrics/domain"
	metricsdto "github.com/example/adnova/internal/metrics/dto"
	"github.com/example/adnova/internal/rules/domain"
	"github.com/shopspring/decimal"
)

func TestEvaluateCampaignKeepsHealthyControlClean(t *testing.T) {
	rules := map[string]domain.AnalysisRule{
		"ROAS_BELOW_TARGET":          {Code: "ROAS_BELOW_TARGET", Severity: "HIGH", Threshold: decimal.RequireFromString("1.3")},
		"CPI_ABOVE_BENCHMARK":        {Code: "CPI_ABOVE_BENCHMARK", Severity: "MEDIUM", Threshold: decimal.RequireFromString("8.5")},
		"PAYER_RATE_BELOW_BENCHMARK": {Code: "PAYER_RATE_BELOW_BENCHMARK", Severity: "HIGH", Threshold: decimal.RequireFromString("0.042")},
		"BUDGET_OVER_90_ROAS_LOW":    {Code: "BUDGET_OVER_90_ROAS_LOW", Severity: "HIGH", Threshold: decimal.RequireFromString("0.9")},
	}
	healthy := metricsdto.CampaignMetric{CampaignID: "google", CampaignName: "Google JP Stable", ROASD7: decimal.RequireFromString("1.43"), CPI: decimal.RequireFromString("5.95"), PayerRate: decimal.RequireFromString("0.056"), BudgetConsumption: decimal.RequireFromString("0.82")}
	if got := evaluateCampaign("tenant", "game", healthy, rules); len(got) != 0 {
		t.Fatalf("healthy control produced %d findings: %#v", len(got), got)
	}
}

func TestConsecutiveD1Decline(t *testing.T) {
	rows := []metricsdomain.CampaignDailyMetric{
		{Date: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), ROASD1: decimal.RequireFromString("0.7")},
		{Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), ROASD1: decimal.RequireFromString("0.9")},
		{Date: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC), ROASD1: decimal.RequireFromString("0.8")},
	}
	declined, values := consecutiveD1Decline(rows, 3)
	if !declined || len(values) != 3 {
		t.Fatalf("got declined=%v values=%v", declined, values)
	}
	rows[2].ROASD1 = decimal.RequireFromString("1.0")
	if declined, _ := consecutiveD1Decline(rows, 3); declined {
		t.Fatal("non-declining series should not match")
	}
}

func TestEvaluateCampaignFindsRisk(t *testing.T) {
	rules := map[string]domain.AnalysisRule{
		"ROAS_BELOW_TARGET":          {Code: "ROAS_BELOW_TARGET", Severity: "HIGH", Threshold: decimal.RequireFromString("1.3")},
		"PAYER_RATE_BELOW_BENCHMARK": {Code: "PAYER_RATE_BELOW_BENCHMARK", Severity: "HIGH", Threshold: decimal.RequireFromString("0.042")},
	}
	risky := metricsdto.CampaignMetric{CampaignID: "meta", CampaignName: "Meta US Growth", ROASD7: decimal.NewFromInt(1), PayerRate: decimal.RequireFromString("0.03")}
	if got := evaluateCampaign("tenant", "game", risky, rules); len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
}
