package service

import (
	"context"
	"testing"
	"time"

	dataqualitydomain "github.com/example/adnova/internal/dataquality/domain"
	"github.com/stretchr/testify/require"
)

type fakeReader struct {
	sources map[string]dataqualitydomain.SourceSnapshot
	imports dataqualitydomain.ImportSummary
}

func (f fakeReader) SourceSnapshot(_ context.Context, _, _, source string) (dataqualitydomain.SourceSnapshot, error) {
	return f.sources[source], nil
}
func (f fakeReader) ImportSummary(context.Context, string, string) (dataqualitydomain.ImportSummary, error) {
	return f.imports, nil
}

func TestAssessReportsMissingAndStaleSources(t *testing.T) {
	recent := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	old := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	service := New(fakeReader{sources: map[string]dataqualitydomain.SourceSnapshot{
		"AD":           {Name: "AD", RowCount: 10, LatestDate: &recent},
		"MMP":          {Name: "MMP", RowCount: 8, LatestDate: &old},
		"GAME_REVENUE": {Name: "GAME_REVENUE"},
		"CREATIVE":     {Name: "CREATIVE", RowCount: 5, LatestDate: &recent},
	}, imports: dataqualitydomain.ImportSummary{Succeeded: 4, Failed: 1}})

	report, err := service.Assess(context.Background(), "tenant-1", "game-1", "2026-08-05")
	require.NoError(t, err)
	require.Equal(t, dataqualitydomain.StatusIncomplete, report.Status)
	require.Equal(t, dataqualitydomain.SourceStale, report.Sources[1].Status)
	require.Equal(t, dataqualitydomain.SourceMissing, report.Sources[2].Status)
	require.Len(t, report.Warnings, 3)
}
