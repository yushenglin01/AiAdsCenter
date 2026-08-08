package agents

import (
	"context"
	"encoding/json"
	"testing"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	dataqualitydomain "github.com/example/adnova/internal/dataquality/domain"
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
