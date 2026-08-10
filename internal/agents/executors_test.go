package agents

import (
	"context"
	"encoding/json"
	"testing"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	"github.com/example/adnova/internal/agent/intelligence"
	creativedomain "github.com/example/adnova/internal/creative/domain"
	dataqualitydomain "github.com/example/adnova/internal/dataquality/domain"
	"github.com/example/adnova/internal/llm"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/stretchr/testify/require"
)

type fakeMetrics struct{ rows int }

func (f fakeMetrics) Recalculate(context.Context, string, string) (int, error) { return f.rows, nil }

type fakeAnalyzer struct{ findings int }

func (f fakeAnalyzer) Analyze(context.Context, string, string) (int, error) { return f.findings, nil }

type fakeProfiler struct{}

func (fakeProfiler) Assess(context.Context, string, string, string) (dataqualitydomain.Report, error) {
	return dataqualitydomain.Report{GameID: "game-1", AnalysisDate: "2026-08-05", Status: dataqualitydomain.StatusValidated, Sources: []dataqualitydomain.SourceSnapshot{}, Warnings: []string{}}, nil
}

type fakeResearch struct{}

func (fakeResearch) SearchVerified(context.Context, string, string, string, string) ([]researchdomain.Evidence, error) {
	return []researchdomain.Evidence{{ID: "source-1", Title: "Verified source", SourceURL: "https://example.com/source"}}, nil
}

type fakeAgentLLM struct{ content json.RawMessage }

func (f fakeAgentLLM) GenerateStructured(context.Context, llm.GenerateRequest) (*llm.GenerateResponse, error) {
	return &llm.GenerateResponse{Content: f.content, Model: "test-model"}, nil
}
func (fakeAgentLLM) Name() string { return "test-provider" }

type fakeCreativeAnalyzer struct{ rows []creativedomain.Finding }

func (f fakeCreativeAnalyzer) Analyze(context.Context, string, string) (int, error) {
	return len(f.rows), nil
}
func (f fakeCreativeAnalyzer) List(context.Context, string, string) ([]creativedomain.Finding, error) {
	return f.rows, nil
}

func agentRuntime(name string, content json.RawMessage) *intelligence.Runtime {
	return intelligence.New(fakeAgentLLM{content: content}, "test-model", map[string]bool{name: true}, nil)
}

func contract(name string) intelligence.Contract {
	return intelligence.Contract{AgentName: name, PromptName: name, PromptVersion: "1.0.0", SchemaVersion: "1.0.0", OutputSchema: `{}`}
}

func TestDataExecutorRecalculatesMetricsAndRules(t *testing.T) {
	payload, _ := json.Marshal(AnalysisPayload{GameID: "game-1", CampaignID: "campaign-1", AnalysisDate: "2026-08-05"})
	executor := NewDataExecutor(fakeMetrics{rows: 12}, fakeAnalyzer{findings: 3}, fakeProfiler{})
	result, err := executor.Execute(context.Background(), agentdomain.AgentInput{TaskID: "step-1", TenantID: "tenant-1", Payload: payload})
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", result.Status)
	require.JSONEq(t, `{"business_rule_findings":3,"calculated_rows":12,"data_quality":{"game_id":"game-1","analysis_date":"2026-08-05","status":"VALIDATED","sources":[],"imports":{"succeeded":0,"failed":0},"warnings":[]},"acquisition_mode":"PREVIOUSLY_IMPORTED_OR_SYNCED","external_connectors":"NOT_INVOKED"}`, string(result.Output))
}

func TestResearchExecutorReturnsOnlyVerifiedRepositoryItems(t *testing.T) {
	payload, _ := json.Marshal(AnalysisPayload{GameID: "game-1", CampaignID: "campaign-1", AnalysisDate: "2026-08-05"})
	result, err := NewResearchExecutor(fakeResearch{}).Execute(context.Background(), agentdomain.AgentInput{TaskID: "step-1", TenantID: "tenant-1", Payload: payload})
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", result.Status)
	require.Contains(t, string(result.Output), "VERIFIED_SOURCES_READY")
	require.Contains(t, string(result.Output), "https://example.com/source")
}

func TestResearchExecutorDoesNotFabricateSources(t *testing.T) {
	result, err := NewResearchExecutor().Execute(context.Background(), agentdomain.AgentInput{TaskID: "step-1"})
	require.NoError(t, err)
	require.Equal(t, "SKIPPED", result.Status)
	require.Contains(t, string(result.Output), "NOT_CONFIGURED")
	require.Contains(t, string(result.Output), `"items":[]`)
}

func TestResearchExecutorAppliesOnlyGroundedLLMSynthesis(t *testing.T) {
	payload, _ := json.Marshal(AnalysisPayload{GameID: "game-1", CampaignID: "campaign-1", AnalysisDate: "2026-08-05"})
	runtime := agentRuntime("research-agent", json.RawMessage(`{"summary":"verified summary","signals":[{"title":"signal","description":"grounded","source_ids":["source-1"],"confidence":0.9}]}`))
	result, err := NewResearchExecutorWithIntelligence(fakeResearch{}, runtime, contract("research-agent")).Execute(context.Background(), agentdomain.AgentInput{TaskID: "step-1", TenantID: "tenant-1", Payload: payload})
	require.NoError(t, err)
	require.Contains(t, string(result.Output), "VERIFIED_SOURCE_LLM_SYNTHESIS")
	require.Contains(t, string(result.Output), "verified summary")
}

func TestResearchExecutorFallsBackWhenLLMInventsSource(t *testing.T) {
	payload, _ := json.Marshal(AnalysisPayload{GameID: "game-1", CampaignID: "campaign-1", AnalysisDate: "2026-08-05"})
	runtime := agentRuntime("research-agent", json.RawMessage(`{"summary":"unsafe","signals":[{"title":"signal","description":"invented","source_ids":["invented-source"],"confidence":1}]}`))
	result, err := NewResearchExecutorWithIntelligence(fakeResearch{}, runtime, contract("research-agent")).Execute(context.Background(), agentdomain.AgentInput{TaskID: "step-1", TenantID: "tenant-1", Payload: payload})
	require.NoError(t, err)
	require.Contains(t, string(result.Output), "DETERMINISTIC_FALLBACK")
	require.NotContains(t, string(result.Output), `"synthesis"`)
}

func TestCreativeExecutorKeepsDeterministicFindingsAuthoritative(t *testing.T) {
	payload, _ := json.Marshal(AnalysisPayload{GameID: "game-1", CampaignID: "campaign-1"})
	analyzer := fakeCreativeAnalyzer{rows: []creativedomain.Finding{{ID: "finding-1", CampaignID: "campaign-1", CreativeID: "creative-1", RuleCode: "CREATIVE_FATIGUE_HIGH"}}}
	runtime := agentRuntime("creative-agent", json.RawMessage(`{"insights":[{"finding_id":"finding-1","interpretation":"解释确定性信号","possible_causes":["待验证假设"],"next_checks":["只读检查"]}]}`))
	result, err := NewCreativeExecutorWithIntelligence(analyzer, runtime, contract("creative-agent")).Execute(context.Background(), agentdomain.AgentInput{TaskID: "step-1", TenantID: "tenant-1", Payload: payload})
	require.NoError(t, err)
	require.Contains(t, string(result.Output), "HYBRID_DETERMINISTIC_LLM")
	require.Contains(t, string(result.Output), "finding-1")
}

func TestCreativeExecutorRejectsUngroundedLLMInsight(t *testing.T) {
	payload, _ := json.Marshal(AnalysisPayload{GameID: "game-1", CampaignID: "campaign-1"})
	analyzer := fakeCreativeAnalyzer{rows: []creativedomain.Finding{{ID: "finding-1", CampaignID: "campaign-1"}}}
	runtime := agentRuntime("creative-agent", json.RawMessage(`{"insights":[{"finding_id":"invented","interpretation":"unsafe","possible_causes":[],"next_checks":[]}]}`))
	result, err := NewCreativeExecutorWithIntelligence(analyzer, runtime, contract("creative-agent")).Execute(context.Background(), agentdomain.AgentInput{TaskID: "step-1", TenantID: "tenant-1", Payload: payload})
	require.NoError(t, err)
	require.Contains(t, string(result.Output), "DETERMINISTIC_FALLBACK")
	require.NotContains(t, string(result.Output), `"insights"`)
}
