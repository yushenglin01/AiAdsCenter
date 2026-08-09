package adjust

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mmpprovider "github.com/example/adnova/internal/mmp/provider"
	"github.com/stretchr/testify/require"
)

func TestFetchNormalizesAdjustReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		require.Equal(t, "app123", r.URL.Query().Get("app_token__in"))
		require.Contains(t, r.URL.Query().Get("metrics"), "purchase_revenue")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rows":[{"day":"2026-08-01","campaign_id_network":"cmp-1","country_code":"US","currency_code":"USD","installs":"2","activation":"2","unique_purchase":"1","purchase_revenue":"3.5"},{"day":"2026-08-01","campaign_id_network":"unknown","country_code":"US","currency_code":"USD","installs":"1","activation":"1","unique_purchase":"1","purchase_revenue":"1"}],"warnings":[],"pagination":null}`))
	}))
	defer server.Close()
	client := New(Config{BaseURL: server.URL, Token: "secret", ActivationMetric: "activation", PayerMetric: "unique_purchase", RevenueMetric: "purchase_revenue", Timeout: time.Second})
	result, err := client.Fetch(context.Background(), mmpprovider.FetchInput{AppID: "app123", From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	require.Equal(t, 2, result.SourceRows)
	require.Equal(t, 1, result.SkippedRows)
	require.Equal(t, "3.500000", result.Records[0]["revenue"])
	require.Equal(t, "1", result.Records[0]["payers"])
}

func TestFetchClassifiesUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer server.Close()
	client := New(Config{BaseURL: server.URL, Token: "secret", ActivationMetric: "activation", PayerMetric: "payer", RevenueMetric: "revenue", Timeout: time.Second})
	_, err := client.Fetch(context.Background(), mmpprovider.FetchInput{AppID: "app123"})
	require.ErrorContains(t, err, "无效")
}

func TestFetchTreatsNoContentAsAuthoritativeEmptyReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	client := New(Config{BaseURL: server.URL, Token: "secret", ActivationMetric: "activation", PayerMetric: "payer", RevenueMetric: "revenue", Timeout: time.Second})
	result, err := client.Fetch(context.Background(), mmpprovider.FetchInput{AppID: "app123"})
	require.NoError(t, err)
	require.Empty(t, result.Records)
}

func TestFetchRejectsNonEmptyReportWithoutMappableCampaign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rows":[{"day":"2026-08-01","campaign_id_network":"unknown","country_code":"US","currency_code":"USD","installs":"1","activation":"1","payer":"0","revenue":"0"}],"warnings":[],"pagination":null}`))
	}))
	defer server.Close()
	client := New(Config{BaseURL: server.URL, Token: "secret", ActivationMetric: "activation", PayerMetric: "payer", RevenueMetric: "revenue", Timeout: time.Second})
	_, err := client.Fetch(context.Background(), mmpprovider.FetchInput{AppID: "app123", From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	require.ErrorContains(t, err, "保留现有指标")
}
