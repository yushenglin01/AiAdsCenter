package appsflyer

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClientFetchAggregatesRawReportsAndRetries(t *testing.T) {
	var installCalls atomic.Int32
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		require.Equal(t, "USD", r.URL.Query().Get("currency"))
		require.Equal(t, "200000", r.URL.Query().Get("maximum_rows"))
		status, body := http.StatusOK, ""
		switch {
		case strings.Contains(r.URL.Path, "/installs_report/"):
			if installCalls.Add(1) == 1 {
				status = http.StatusTooManyRequests
				break
			}
			body = "Install Time,Campaign ID,Country Code\n2026-08-01 01:00:00,cmp-1,US\n2026-08-01 02:00:00,cmp-1,US\n2026-08-01 03:00:00,,US\n"
		case strings.Contains(r.URL.Path, "/in_app_events_report/"):
			require.Equal(t, "af_purchase,first_deposit", r.URL.Query().Get("event_name"))
			body = "Event Time,Campaign ID,Country Code,AppsFlyer ID,Event Revenue USD\n2026-08-01 05:00:00,cmp-1,US,user-1,3.50\n2026-08-01 06:00:00,cmp-1,US,user-1,1.50\n2026-08-01 07:00:00,cmp-1,US,user-2,2.00\n"
		default:
			status = http.StatusNotFound
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})

	client := New(Config{BaseURL: "https://hq1.appsflyer.test", Token: "secret-token", Timeout: time.Second, MaxRetries: 1, PurchaseEvents: []string{"af_purchase", "first_deposit"}})
	client.httpClient.Transport = transport
	client.wait = func(context.Context, time.Duration) error { return nil }
	result, err := client.Fetch(context.Background(), FetchInput{AppID: "com.example.game", From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	require.Equal(t, int32(2), installCalls.Load())
	require.Equal(t, 6, result.SourceRows)
	require.Equal(t, 1, result.SkippedRows)
	require.Len(t, result.Records, 1)
	require.Equal(t, "2", result.Records[0]["installs"])
	require.Equal(t, "2", result.Records[0]["activations"])
	require.Equal(t, "2", result.Records[0]["payers"])
	require.Equal(t, "7.000000", result.Records[0]["revenue"])
	require.Contains(t, result.WarningMessage, "1 行缺少 campaign_id")
}

func TestClientDoesNotExposeTokenInProviderError(t *testing.T) {
	client := New(Config{BaseURL: "https://hq1.appsflyer.test", Token: "do-not-leak", Timeout: time.Second, PurchaseEvents: []string{"af_purchase"}})
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("token do-not-leak")), Request: r}, nil
	})
	_, err := client.Fetch(context.Background(), FetchInput{AppID: "app", From: time.Now(), To: time.Now()})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "do-not-leak")
	var providerErr *ProviderError
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "UNAUTHORIZED", providerErr.Code)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestNormalizeRejectsNegativeRevenue(t *testing.T) {
	_, err := normalize(nil, []map[string]string{{"event_time": "2026-08-01 00:00:00", "campaign_id": "cmp", "country_code": "US", "appsflyer_id": "user", "event_revenue_usd": "-1"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "负数收入")
}

func TestNormalizeRejectsRowsWithoutCampaignInsteadOfReplacingWithEmptyData(t *testing.T) {
	_, err := normalize([]map[string]string{{"install_time": "2026-08-01 00:00:00", "country_code": "US"}}, nil)
	require.Error(t, err)
	var providerErr *ProviderError
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "NO_MAPPABLE_ROWS", providerErr.Code)
}

func TestClassifyAppsFlyerCallLimitAsRetryable(t *testing.T) {
	code, _, retry := classifyStatus(http.StatusBadRequest, `{"error":"CallLimit"}`)
	require.Equal(t, "RATE_LIMITED", code)
	require.True(t, retry)
}
