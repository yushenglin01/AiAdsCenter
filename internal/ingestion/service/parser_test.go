package service

import (
	"testing"

	"github.com/example/adnova/pkg/provider"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestParseAdRowPrecision(t *testing.T) {
	record := provider.Record{"date": "2026-07-01", "campaign_external_id": "meta-us-001", "country": "us", "currency": "usd", "spend": "10.123456", "impressions": "100", "clicks": "5", "installs": "2"}
	row, err := parseCommon(record)
	require.NoError(t, err)
	require.NoError(t, parseByType(ImportAd, record, &row))
	require.True(t, row.spend.Equal(decimal.RequireFromString("10.123456")))
	require.Equal(t, "US", row.country)
	require.Equal(t, "USD", row.currency)
}

func TestParseRowsRejectInvalidValues(t *testing.T) {
	tests := []struct{ name, key, value string }{
		{name: "negative spend", key: "spend", value: "-1"},
		{name: "spend precision overflow", key: "spend", value: "1.1234567"},
		{name: "spend range overflow", key: "spend", value: "100000000000000"},
		{name: "fractional installs", key: "installs", value: "1.5"},
		{name: "missing date", key: "date", value: ""},
	}
	base := provider.Record{"date": "2026-07-01", "campaign_external_id": "meta-us-001", "country": "US", "currency": "USD", "spend": "1", "impressions": "100", "clicks": "5", "installs": "2"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := provider.Record{}
			for key, value := range base {
				record[key] = value
			}
			record[tt.key] = tt.value
			row, err := parseCommon(record)
			if err == nil {
				err = parseByType(ImportAd, record, &row)
			}
			require.Error(t, err)
		})
	}
}

func TestFormulaInjectionGuard(t *testing.T) {
	require.True(t, unsafeText("=HYPERLINK('bad')"))
	require.True(t, unsafeText("@SUM(A1)"))
	require.False(t, unsafeText("meta-us-001"))
}

func TestSourceAllowList(t *testing.T) {
	require.True(t, sourceAllowed(ImportAd, "META"))
	require.True(t, sourceAllowed(ImportAd, "INTERNAL"))
	require.True(t, sourceAllowed(ImportMMP, "APPSFLYER"))
	require.False(t, sourceAllowed(ImportAd, "APPSFLYER"))
}

func TestFrequencyUsesDatabaseColumnRange(t *testing.T) {
	_, err := parseDecimal(provider.Record{"frequency": "1000000"}, "frequency")
	require.ErrorContains(t, err, "DECIMAL(12,6)")

	value, err := parseDecimal(provider.Record{"frequency": "999999.999999"}, "frequency")
	require.NoError(t, err)
	require.True(t, value.Equal(maxFrequencyDecimal))
}
