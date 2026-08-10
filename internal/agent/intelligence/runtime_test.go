package intelligence

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	"github.com/example/adnova/internal/llm"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	name     string
	response *llm.GenerateResponse
	err      error
}

func (f fakeClient) GenerateStructured(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return f.response, f.err
}

func (f fakeClient) Name() string { return f.name }

type fakeRecorder struct {
	rows []*agentdomain.ModelUsageRecord
}

func (f *fakeRecorder) Create(_ context.Context, row *agentdomain.ModelUsageRecord) error {
	f.rows = append(f.rows, row)
	return nil
}
func (f *fakeRecorder) UpdateStatus(_ context.Context, id, status string) error {
	for _, row := range f.rows {
		if row.ID == id {
			row.Status = status
		}
	}
	return nil
}

func TestRuntimeRecordsNormalizedUsage(t *testing.T) {
	recorder := &fakeRecorder{}
	runtime := New(fakeClient{name: "deepseek", response: &llm.GenerateResponse{
		Content: json.RawMessage(`{"ok":true}`), Model: "deepseek-model",
		Usage: llm.Usage{InputTokens: 12, OutputTokens: 4}, Latency: 20 * time.Millisecond,
	}}, "configured-model", map[string]bool{"research-agent": true}, recorder)
	result, err := runtime.Generate(context.Background(), Request{
		AgentName: "research-agent", TenantID: "tenant-1", TaskID: "task-1",
		Contract:  Contract{PromptName: "research_agent_llm", PromptVersion: "1.0.0", SchemaVersion: "1.0.0"},
		InputJSON: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	require.Equal(t, "deepseek", result.Provider)
	require.Len(t, recorder.rows, 1)
	require.Equal(t, "SUCCEEDED", recorder.rows[0].Status)
	require.Equal(t, 12, recorder.rows[0].InputTokens)
	require.Equal(t, "research_agent_llm", recorder.rows[0].PromptName)
	runtime.MarkValidationFailed(context.Background(), result)
	require.Equal(t, "VALIDATION_FAILED", recorder.rows[0].Status)
}

func TestRuntimeRecordsSafeFailureAndDisablesMock(t *testing.T) {
	recorder := &fakeRecorder{}
	runtime := New(fakeClient{name: "openai", err: &llm.ClientError{Category: llm.ErrorRateLimit, Message: "rate limited", Retryable: true}}, "model", map[string]bool{"creative-agent": true}, recorder)
	_, err := runtime.Generate(context.Background(), Request{AgentName: "creative-agent", TenantID: "tenant-1", TaskID: "task-1", Contract: Contract{PromptName: "creative", PromptVersion: "1", SchemaVersion: "1"}})
	require.Error(t, err)
	require.Equal(t, "RATE_LIMIT", FailureCategory(err))
	require.Len(t, recorder.rows, 1)
	require.Equal(t, "FAILED", recorder.rows[0].Status)

	mockRuntime := New(fakeClient{name: "mock"}, "mock-business-v1", map[string]bool{"creative-agent": true}, nil)
	require.False(t, mockRuntime.Enabled("creative-agent"))
}
