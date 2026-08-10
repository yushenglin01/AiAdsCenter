package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OpenAICompatibleClient struct {
	provider string
	baseURL  string
	apiKey   string
	model    string
	client   *http.Client
	retries  int
}

func NewOpenAICompatible(baseURL, apiKey, model string, timeout time.Duration) (*OpenAICompatibleClient, error) {
	return NewOpenAICompatibleForProvider("openai-compatible", baseURL, apiKey, model, timeout)
}

func NewOpenAICompatibleForProvider(provider, baseURL, apiKey, model string, timeout time.Duration) (*OpenAICompatibleClient, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(apiKey) == "" || strings.TrimSpace(model) == "" {
		return nil, &ClientError{Category: ErrorConfiguration, Message: "OpenAI-compatible base URL, API key and model are required"}
	}
	if provider == "" {
		return nil, &ClientError{Category: ErrorConfiguration, Message: "LLM provider name is required"}
	}
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil || parsedBaseURL.Host == "" || (parsedBaseURL.Scheme != "https" && parsedBaseURL.Scheme != "http") || parsedBaseURL.User != nil || parsedBaseURL.RawQuery != "" || parsedBaseURL.Fragment != "" {
		return nil, &ClientError{Category: ErrorConfiguration, Message: "OpenAI-compatible base URL must be an absolute HTTP(S) URL without credentials, query or fragment"}
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &OpenAICompatibleClient{provider: provider, baseURL: baseURL, apiKey: apiKey, model: model, client: &http.Client{Timeout: timeout}, retries: 2}, nil
}

func (c *OpenAICompatibleClient) Name() string { return c.provider }

func (c *OpenAICompatibleClient) GenerateStructured(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	started := time.Now()
	model := req.Model
	if model == "" {
		model = c.model
	}
	userContent := req.UserPrompt + "\n\nSTRUCTURED INPUT:\n" + string(req.InputJSON) + "\n\nOUTPUT SCHEMA:\n" + req.OutputSchema
	payload := map[string]any{"model": model, "messages": []map[string]string{{"role": "system", "content": req.SystemPrompt}, {"role": "user", "content": userContent}}, "temperature": req.Temperature, "max_tokens": req.MaxTokens, "response_format": map[string]string{"type": "json_object"}}
	body, _ := json.Marshal(payload)
	for attempt := 0; attempt <= c.retries; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return nil, &ClientError{Category: ErrorInvalidRequest, Message: "create provider request failed"}
		}
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
		request.Header.Set("Content-Type", "application/json")
		if req.TraceID != "" {
			request.Header.Set("X-Trace-ID", req.TraceID)
		}
		response, err := c.client.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				return nil, &ClientError{Category: ErrorTimeout, Message: "model request timed out", Retryable: true}
			}
			if attempt < c.retries {
				continue
			}
			return nil, &ClientError{Category: ErrorUnavailable, Message: "model provider unavailable", Retryable: true}
		}
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		_ = response.Body.Close()
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			return nil, &ClientError{Category: ErrorAuthentication, Message: "model provider authentication failed", Retryable: false, Code: fmt.Sprint(response.StatusCode)}
		}
		if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
			if attempt < c.retries {
				continue
			}
			return nil, &ClientError{Category: map[bool]ErrorCategory{true: ErrorRateLimit, false: ErrorUnavailable}[response.StatusCode == http.StatusTooManyRequests], Message: "model provider temporarily unavailable", Retryable: true, Code: fmt.Sprint(response.StatusCode)}
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, &ClientError{Category: ErrorInvalidRequest, Message: "model provider rejected request", Retryable: false, Code: fmt.Sprint(response.StatusCode)}
		}
		var decoded struct {
			Model   string `json:"model"`
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(responseBody, &decoded) != nil || len(decoded.Choices) == 0 || !json.Valid([]byte(decoded.Choices[0].Message.Content)) {
			return nil, &ClientError{Category: ErrorMalformed, Message: "model provider returned malformed structured JSON", Retryable: false}
		}
		responseModel := strings.TrimSpace(decoded.Model)
		if responseModel == "" {
			responseModel = model
		}
		return &GenerateResponse{Content: json.RawMessage(decoded.Choices[0].Message.Content), Model: responseModel, FinishReason: decoded.Choices[0].FinishReason, Usage: Usage{InputTokens: decoded.Usage.PromptTokens, OutputTokens: decoded.Usage.CompletionTokens}, Latency: time.Since(started)}, nil
	}
	return nil, &ClientError{Category: ErrorUnavailable, Message: "model provider unavailable", Retryable: true}
}
