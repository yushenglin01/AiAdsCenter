package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/example/adnova/internal/agent/intelligence"
	"github.com/example/adnova/internal/llm"
	"github.com/stretchr/testify/require"
)

type fakeReportLLM struct{ content json.RawMessage }

func (f fakeReportLLM) GenerateStructured(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return &llm.GenerateResponse{Content: f.content, Model: "model"}, nil
}
func (fakeReportLLM) Name() string { return "test-provider" }

func TestLLMPolisherRejectsDigestMismatch(t *testing.T) {
	runtime := intelligence.New(fakeReportLLM{content: json.RawMessage(`{"summary":"changed","source_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)}, "model", map[string]bool{"report-agent": true}, nil)
	polisher := NewLLMPolisher(runtime, intelligence.Contract{PromptVersion: "1", SchemaVersion: "1", OutputSchema: `{}`})
	_, err := polisher.Polish(context.Background(), Input{TenantID: "tenant-1", TaskID: "task-1"}, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	require.ErrorContains(t, err, "digest mismatch")
}
