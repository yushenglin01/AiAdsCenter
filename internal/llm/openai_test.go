package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAICompatibleNormalizesSuccess(t *testing.T) {
	client, err := NewOpenAICompatibleForProvider("deepseek", "https://provider.example/v1/", "secret-value", "demo-model", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://provider.example/v1/chat/completions" {
			t.Errorf("unexpected request URL: %s", r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer secret-value" {
			t.Error("missing auth header")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["model"] != "demo-model" {
			t.Errorf("unexpected model: %v", payload["model"])
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
	if client.Name() != "deepseek" {
		t.Fatalf("unexpected provider name: %s", client.Name())
	}
}

func TestOpenAICompatibleFallsBackToRequestedModel(t *testing.T) {
	client, err := NewOpenAICompatibleForProvider("openai", "https://api.openai.com/v1", "secret-value", "configured-model", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.client = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"{}"},"finish_reason":"stop"}],"usage":{}}`))}, nil
	})}
	response, err := client.GenerateStructured(context.Background(), GenerateRequest{Model: "request-model", InputJSON: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if response.Model != "request-model" {
		t.Fatalf("model=%q", response.Model)
	}
}

func TestOpenAICompatibleRejectsUnsafeBaseURL(t *testing.T) {
	_, err := NewOpenAICompatibleForProvider("custom", "https://user:pass@provider.example/v1", "secret-value", "model", time.Second)
	if err == nil {
		t.Fatal("expected invalid base URL error")
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
