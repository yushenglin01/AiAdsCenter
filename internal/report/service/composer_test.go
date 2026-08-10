package service

import (
	"context"
	"errors"
	"testing"
	"time"

	businessdomain "github.com/example/adnova/internal/business/domain"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/stretchr/testify/require"
)

type fakePolisher struct {
	result *PolishedSummary
	err    error
	digest string
}

func (f *fakePolisher) Enabled() bool { return true }
func (f *fakePolisher) Polish(_ context.Context, _ Input, digest string) (*PolishedSummary, error) {
	f.digest = digest
	return f.result, f.err
}

func TestComposeIncludesProvenanceAndVerifiedSources(t *testing.T) {
	report := Compose(Input{TenantID: "tenant-1", TaskID: "task-1", WorkflowID: "workflow-1", Result: businessdomain.Result{Summary: "summary"}, Research: []researchdomain.Evidence{{ID: "source-1", Title: "Market report", SourceURL: "https://example.com/report", Publisher: "Example", PublishedAt: time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)}}})
	require.Equal(t, GeneratorAgent, report.GeneratorAgent)
	require.Len(t, report.SourceDigest, 64)
	require.Contains(t, report.ContentMarkdown, "https://example.com/report")
	require.Contains(t, string(report.ProvenanceJSON), "source-1")
}

func TestComposeAppliesOnlyGuardedSummaryPolish(t *testing.T) {
	input := Input{TenantID: "tenant-1", TaskID: "task-1", WorkflowID: "workflow-1", Result: businessdomain.Result{Summary: "original"}}
	polisher := &fakePolisher{result: &PolishedSummary{Summary: "polished", Provider: "openai", Model: "model", PromptVersion: "1.0.0", SchemaVersion: "1.0.0"}}
	report := ComposeWithPolisher(context.Background(), input, polisher)
	require.Equal(t, "polished", report.Summary)
	require.Equal(t, report.SourceDigest, polisher.digest)
	require.Contains(t, string(report.ProvenanceJSON), `"status":"APPLIED"`)
	require.Contains(t, report.ContentMarkdown, "polished")
}

func TestComposeFallsBackWithoutChangingFactsWhenPolishFails(t *testing.T) {
	input := Input{TenantID: "tenant-1", TaskID: "task-1", Result: businessdomain.Result{Summary: "original", Findings: []businessdomain.Finding{{Conclusion: "fact", Severity: "HIGH", Description: "deterministic"}}}}
	report := ComposeWithPolisher(context.Background(), input, &fakePolisher{err: errors.New("provider failed")})
	require.Equal(t, "original", report.Summary)
	require.Contains(t, report.ContentMarkdown, "deterministic")
	require.Contains(t, string(report.ProvenanceJSON), `"status":"FALLBACK"`)
}
