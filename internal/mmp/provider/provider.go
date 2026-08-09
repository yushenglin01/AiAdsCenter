package provider

import (
	"context"
	"time"

	dataprovider "github.com/example/adnova/pkg/provider"
)

type FetchInput struct {
	AppID string
	From  time.Time
	To    time.Time
}

type FetchResult struct {
	Records        []dataprovider.Record
	SourceRows     int
	SkippedRows    int
	WarningMessage string
}

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

type Fetcher interface {
	Configured() bool
	Fetch(context.Context, FetchInput) (*FetchResult, error)
}
