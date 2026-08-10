package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/example/adnova/internal/agent/intelligence"
)

type PolishedSummary struct {
	Summary       string
	Provider      string
	Model         string
	PromptVersion string
	SchemaVersion string
}

type Polisher interface {
	Enabled() bool
	Polish(context.Context, Input, string) (*PolishedSummary, error)
}

type LLMPolisher struct {
	runtime  *intelligence.Runtime
	contract intelligence.Contract
}

func NewLLMPolisher(runtime *intelligence.Runtime, contract intelligence.Contract) *LLMPolisher {
	return &LLMPolisher{runtime: runtime, contract: contract}
}

func (p *LLMPolisher) Enabled() bool {
	return p != nil && p.runtime.Enabled("report-agent")
}

func (p *LLMPolisher) Polish(ctx context.Context, input Input, digest string) (*PolishedSummary, error) {
	payload, _ := json.Marshal(map[string]any{
		"summary": input.Result.Summary, "findings": input.Result.Findings,
		"recommendations": input.Result.Recommendations, "source_digest": digest,
	})
	generated, err := p.runtime.Generate(ctx, intelligence.Request{
		AgentName: "report-agent", TenantID: input.TenantID, TaskID: input.TaskID, TraceID: input.TraceID,
		Contract: p.contract, UserPrompt: "在不改变任何事实的前提下润色摘要，并原样返回 source_digest。",
		InputJSON: payload, Temperature: 0.1, MaxTokens: 800,
	})
	if err != nil {
		return nil, err
	}
	var output struct {
		Summary      string `json:"summary"`
		SourceDigest string `json:"source_digest"`
	}
	if err := json.Unmarshal(generated.Content, &output); err != nil {
		p.runtime.MarkValidationFailed(ctx, generated)
		return nil, fmt.Errorf("decode report LLM output: %w", err)
	}
	output.Summary = strings.TrimSpace(output.Summary)
	if output.SourceDigest != digest {
		p.runtime.MarkValidationFailed(ctx, generated)
		return nil, fmt.Errorf("report LLM source digest mismatch")
	}
	if output.Summary == "" || utf8.RuneCountInString(output.Summary) > 2000 {
		p.runtime.MarkValidationFailed(ctx, generated)
		return nil, fmt.Errorf("report LLM summary length is invalid")
	}
	return &PolishedSummary{
		Summary: output.Summary, Provider: generated.Provider, Model: generated.Model,
		PromptVersion: generated.PromptVersion, SchemaVersion: generated.SchemaVersion,
	}, nil
}

func FailureCategory(err error) string { return intelligence.FailureCategory(err) }
