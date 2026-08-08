package service

import (
	"testing"

	"github.com/example/adnova/internal/metrics/domain"
	"github.com/shopspring/decimal"
)

func TestCalculate(t *testing.T) {
	input := domain.MetricInput{
		Spend:         decimal.NewFromInt(100),
		DailyBudget:   decimal.NewFromInt(125),
		RevenueD1:     decimal.NewFromInt(50),
		RevenueD3:     decimal.NewFromInt(90),
		RevenueD7:     decimal.NewFromInt(150),
		Impressions:   10000,
		Clicks:        500,
		Installs:      100,
		Registrations: 80,
		Payers:        20,
	}
	got := Calculate(input)
	wants := map[string]struct{ got, want decimal.Decimal }{
		"ctr":                {got.CTR, decimal.RequireFromString("0.05")},
		"cvr":                {got.CVR, decimal.RequireFromString("0.2")},
		"cpi":                {got.CPI, decimal.NewFromInt(1)},
		"cpa":                {got.CPA, decimal.NewFromInt(5)},
		"payer_rate":         {got.PayerRate, decimal.RequireFromString("0.25")},
		"roas_d1":            {got.ROASD1, decimal.RequireFromString("0.5")},
		"roas_d3":            {got.ROASD3, decimal.RequireFromString("0.9")},
		"roas_d7":            {got.ROASD7, decimal.RequireFromString("1.5")},
		"budget_consumption": {got.BudgetConsumption, decimal.RequireFromString("0.8")},
		"ltv_d7":             {got.LTVD7, decimal.RequireFromString("1.875")},
	}
	for name, item := range wants {
		t.Run(name, func(t *testing.T) {
			if !item.got.Equal(item.want) {
				t.Fatalf("got %s, want %s", item.got, item.want)
			}
		})
	}
}

func TestCalculateZeroDenominators(t *testing.T) {
	got := Calculate(domain.MetricInput{RevenueD7: decimal.NewFromInt(10)})
	values := []decimal.Decimal{got.CTR, got.CVR, got.CPI, got.CPA, got.PayerRate, got.ROASD1, got.ROASD3, got.ROASD7, got.BudgetConsumption, got.LTVD7}
	for i, value := range values {
		if !value.IsZero() {
			t.Fatalf("value %d = %s, want 0", i, value)
		}
	}
}

func TestCalculateRoundsToEightPlaces(t *testing.T) {
	got := Calculate(domain.MetricInput{Clicks: 1, Impressions: 3})
	if got.CTR.StringFixed(8) != "0.33333333" {
		t.Fatalf("got %s", got.CTR)
	}
}
