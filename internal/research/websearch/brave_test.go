package websearch

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestBraveSearchNormalizesHTTPSResults(t *testing.T) {
	client := NewBrave(BraveConfig{BaseURL: "https://api.search.brave.com", APIKey: "secret", Timeout: time.Second, MaxResults: 8, SafeSearch: "strict", ImportEnabled: true})
	client.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "secret", r.Header.Get("X-Subscription-Token"))
		require.Equal(t, "game advertising", r.URL.Query().Get("q"))
		require.Equal(t, "strict", r.URL.Query().Get("safe_search"))
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"web":{"results":[{"title":"Fresh report","url":"https://Example.com/report","description":"Current market signal","profile":{"long_name":"Example Research"},"page_age":"2026-08-10T02:00:00Z","language":"en"},{"title":"Unsafe","url":"http://example.com","description":"ignored"}]}}`))}, nil
	})
	response, err := client.Search(context.Background(), Query{Text: "game advertising", Count: 5})
	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	require.Equal(t, "Example Research", response.Results[0].Publisher)
	require.Equal(t, "https://Example.com/report", response.Results[0].URL)
}

func TestBraveSearchReportsMissingCredential(t *testing.T) {
	client := NewBrave(BraveConfig{BaseURL: "https://api.search.brave.com", Timeout: time.Second, MaxResults: 8, SafeSearch: "strict"})
	require.False(t, client.Capability().Configured)
	_, err := client.Search(context.Background(), Query{Text: "test"})
	var providerErr *Error
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "NOT_CONFIGURED", providerErr.Code)
}

func TestBraveSearchClassifiesRateLimit(t *testing.T) {
	client := NewBrave(BraveConfig{BaseURL: "https://api.search.brave.com", APIKey: "secret", Timeout: time.Second, MaxResults: 8, SafeSearch: "strict"})
	client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusTooManyRequests, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	_, err := client.Search(context.Background(), Query{Text: "test"})
	var providerErr *Error
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "RATE_LIMITED", providerErr.Code)
	require.True(t, providerErr.Retryable)
}
