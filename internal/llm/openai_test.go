package llm

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAICompatibleNormalizesSuccess(t *testing.T) {
	client, err := NewOpenAICompatible("https://provider.example/v1", "secret-value", "demo-model", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer secret-value" {
			t.Error("missing auth header")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"model":"demo-model","choices":[{"message":{"content":"{\"status\":\"ok\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":5}}`))}, nil
	})}
	response, err := client.GenerateStructured(context.Background(), GenerateRequest{InputJSON: []byte(`{}`), OutputSchema: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	if response.Model != "demo-model" || response.Usage.InputTokens != 12 || string(response.Content) != `{"status":"ok"}` {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestOpenAICompatibleRetriesUnavailableWithoutLeakingSecret(t *testing.T) {
	var calls int32
	client, _ := NewOpenAICompatible("https://provider.example/v1", "secret-value", "model", time.Second)
	client.client = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("secret-value provider error"))}, nil
	})}
	_, err := client.GenerateStructured(context.Background(), GenerateRequest{InputJSON: []byte(`{}`)})
	if err == nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	if strings.Contains(err.Error(), "secret-value") {
		t.Fatal("client error leaked credential")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
