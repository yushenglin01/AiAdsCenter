package service

import (
	"github.com/example/adnova/internal/metrics/domain"
	"github.com/shopspring/decimal"
)

const metricPrecision = 8

func Calculate(input domain.MetricInput) domain.MetricValues {
	return domain.MetricValues{
		CTR:               safeDiv(decimal.NewFromInt(input.Clicks), decimal.NewFromInt(input.Impressions)),
		CVR:               safeDiv(decimal.NewFromInt(input.Installs), decimal.NewFromInt(input.Clicks)),
		CPI:               safeDiv(input.Spend, decimal.NewFromInt(input.Installs)),
		CPA:               safeDiv(input.Spend, decimal.NewFromInt(input.Payers)),
		PayerRate:         safeDiv(decimal.NewFromInt(input.Payers), decimal.NewFromInt(input.Registrations)),
		ROASD1:            safeDiv(input.RevenueD1, input.Spend),
		ROASD3:            safeDiv(input.RevenueD3, input.Spend),
		ROASD7:            safeDiv(input.RevenueD7, input.Spend),
		BudgetConsumption: safeDiv(input.Spend, input.DailyBudget),
		LTVD7:             safeDiv(input.RevenueD7, decimal.NewFromInt(input.Registrations)),
	}
}

func safeDiv(numerator, denominator decimal.Decimal) decimal.Decimal {
	if denominator.IsZero() {
		return decimal.Zero
	}
	return numerator.DivRound(denominator, metricPrecision)
}
