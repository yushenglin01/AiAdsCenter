package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/example/adnova/internal/agent/intelligence"
	"github.com/example/adnova/internal/llm"
	"github.com/stretchr/testify/require"
)

type fakeOpenClawLLM struct{ content json.RawMessage }

func (f fakeOpenClawLLM) GenerateStructured(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return &llm.GenerateResponse{Content: f.content, Model: "intent-model"}, nil
}
func (fakeOpenClawLLM) Name() string { return "test-provider" }

func TestLLMParserReturnsValidatedProvenance(t *testing.T) {
	content := json.RawMessage(`{"intent":"GET_WORKFLOW_STATUS","input":{"workflow_id":"workflow-1"},"requires_confirmation":false,"confidence":0.95,"missing_fields":[]}`)
	runtime := intelligence.New(fakeOpenClawLLM{content: content}, "intent-model", map[string]bool{"openclaw-agent": true}, nil)
	parser := NewLLMParser(runtime, intelligence.Contract{PromptVersion: "1.0.0", SchemaVersion: "1.0.0", OutputSchema: `{}`})
	parsed, err := parser.Parse(context.Background(), Actor{TenantID: "tenant-1"}, "查询 workflow-1")
	require.NoError(t, err)
	require.Equal(t, IntentGetWorkflowStatus, parsed.Intent)
	require.Equal(t, "test-provider", parsed.Provider)
	require.False(t, parsed.RequiresConfirmation)
}

func TestLLMParserRejectsUnknownOutputFields(t *testing.T) {
	content := json.RawMessage(`{"intent":"LIST_NOTIFICATIONS","input":{},"requires_confirmation":false,"confidence":0.95,"missing_fields":[],"execute_now":true}`)
	runtime := intelligence.New(fakeOpenClawLLM{content: content}, "intent-model", map[string]bool{"openclaw-agent": true}, nil)
	parser := NewLLMParser(runtime, intelligence.Contract{OutputSchema: `{}`})
	_, err := parser.Parse(context.Background(), Actor{TenantID: "tenant-1"}, "查看通知")
	require.ErrorContains(t, err, "failed validation")
}
