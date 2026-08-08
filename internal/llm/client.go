package llm

import (
	"context"
	"encoding/json"
	"time"
)

type GenerateRequest struct {
	Model        string          `json:"model"`
	SystemPrompt string          `json:"system_prompt"`
	UserPrompt   string          `json:"user_prompt"`
	InputJSON    json.RawMessage `json:"input_json"`
	OutputSchema string          `json:"output_schema"`
	Temperature  float64         `json:"temperature"`
	MaxTokens    int             `json:"max_tokens"`
	Timeout      time.Duration   `json:"timeout"`
	TraceID      string          `json:"trace_id"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type GenerateResponse struct {
	Content      json.RawMessage `json:"content"`
	Model        string          `json:"model"`
	FinishReason string          `json:"finish_reason"`
	Usage        Usage           `json:"usage"`
	Latency      time.Duration   `json:"latency"`
}

type Client interface {
	GenerateStructured(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	Name() string
}

type ErrorCategory string

const (
	ErrorInvalidRequest ErrorCategory = "INVALID_REQUEST"
	ErrorConfiguration  ErrorCategory = "CONFIGURATION"
	ErrorAuthentication ErrorCategory = "AUTHENTICATION"
	ErrorRateLimit      ErrorCategory = "RATE_LIMIT"
	ErrorUnavailable    ErrorCategory = "UNAVAILABLE"
	ErrorTimeout        ErrorCategory = "TIMEOUT"
	ErrorMalformed      ErrorCategory = "MALFORMED_RESPONSE"
)

type ClientError struct {
	Category  ErrorCategory
	Message   string
	Retryable bool
	Code      string
}

func (e *ClientError) Error() string { return e.Message }
