package service

import (
	"context"
	"fmt"
	"time"

	dataqualitydomain "github.com/example/adnova/internal/dataquality/domain"
)

type Reader interface {
	SourceSnapshot(context.Context, string, string, string) (dataqualitydomain.SourceSnapshot, error)
	ImportSummary(context.Context, string, string) (dataqualitydomain.ImportSummary, error)
}

type Service struct {
	reader    Reader
	staleDays int
}

func New(reader Reader) *Service { return &Service{reader: reader, staleDays: 3} }

func (s *Service) Assess(ctx context.Context, tenantID, gameID, analysisDate string) (dataqualitydomain.Report, error) {
	through, err := time.Parse("2006-01-02", analysisDate)
	if err != nil {
		return dataqualitydomain.Report{}, fmt.Errorf("analysis_date must be YYYY-MM-DD")
	}
	result := dataqualitydomain.Report{GameID: gameID, AnalysisDate: analysisDate, Status: dataqualitydomain.StatusValidated, Warnings: []string{}}
	missing, stale := false, false
	for _, name := range []string{"AD", "MMP", "GAME_REVENUE", "CREATIVE"} {
		snapshot, readErr := s.reader.SourceSnapshot(ctx, tenantID, gameID, name)
		if readErr != nil {
			return dataqualitydomain.Report{}, readErr
		}
		snapshot.Status = dataqualitydomain.SourceReady
		if snapshot.RowCount == 0 || snapshot.LatestDate == nil {
			snapshot.Status, missing = dataqualitydomain.SourceMissing, true
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s data is missing", name))
		} else if snapshot.LatestDate.Before(through.AddDate(0, 0, -s.staleDays)) {
			snapshot.Status, stale = dataqualitydomain.SourceStale, true
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s data is older than %d days", name, s.staleDays))
		}
		result.Sources = append(result.Sources, snapshot)
	}
	result.Imports, err = s.reader.ImportSummary(ctx, tenantID, gameID)
	if err != nil {
		return dataqualitydomain.Report{}, err
	}
	if result.Imports.Failed > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d import jobs failed", result.Imports.Failed))
	}
	if missing {
		result.Status = dataqualitydomain.StatusIncomplete
	} else if stale {
		result.Status = dataqualitydomain.StatusStale
	}
	return result, nil
}
