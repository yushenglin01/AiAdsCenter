package domain

import "time"

const (
	StatusValidated  = "VALIDATED"
	StatusIncomplete = "INCOMPLETE"
	StatusStale      = "STALE"

	SourceReady   = "READY"
	SourceMissing = "MISSING"
	SourceStale   = "STALE"
)

type SourceSnapshot struct {
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	RowCount      int64      `json:"row_count"`
	CampaignCount int64      `json:"campaign_count"`
	LatestDate    *time.Time `json:"latest_date,omitempty"`
}

type ImportSummary struct {
	Succeeded int64 `json:"succeeded"`
	Failed    int64 `json:"failed"`
}

type Report struct {
	GameID       string           `json:"game_id"`
	AnalysisDate string           `json:"analysis_date"`
	Status       string           `json:"status"`
	Sources      []SourceSnapshot `json:"sources"`
	Imports      ImportSummary    `json:"imports"`
	Warnings     []string         `json:"warnings"`
}
