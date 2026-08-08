package service

import (
	"testing"
	"time"

	businessdomain "github.com/example/adnova/internal/business/domain"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/stretchr/testify/require"
)

func TestComposeIncludesProvenanceAndVerifiedSources(t *testing.T) {
	report := Compose(Input{TenantID: "tenant-1", TaskID: "task-1", WorkflowID: "workflow-1", Result: businessdomain.Result{Summary: "summary"}, Research: []researchdomain.Evidence{{ID: "source-1", Title: "Market report", SourceURL: "https://example.com/report", Publisher: "Example", PublishedAt: time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)}}})
	require.Equal(t, GeneratorAgent, report.GeneratorAgent)
	require.Len(t, report.SourceDigest, 64)
	require.Contains(t, report.ContentMarkdown, "https://example.com/report")
	require.Contains(t, string(report.ProvenanceJSON), "source-1")
}
