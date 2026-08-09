package websearch

import (
	"context"
	"time"
)

type Query struct {
	Text       string
	Count      int
	Country    string
	SearchLang string
	Freshness  string
}

type Result struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Publisher   string `json:"publisher"`
	PublishedAt string `json:"published_at,omitempty"`
	Language    string `json:"language,omitempty"`
}

type Response struct {
	Provider   string    `json:"provider"`
	Results    []Result  `json:"results"`
	SearchedAt time.Time `json:"searched_at"`
}

type Capability struct {
	Configured    bool     `json:"configured"`
	Provider      string   `json:"provider"`
	ImportEnabled bool     `json:"import_enabled"`
	MaxResults    int      `json:"max_results"`
	Details       []string `json:"details"`
}

type Provider interface {
	Search(context.Context, Query) (Response, error)
	Capability() Capability
}

type Error struct {
	Code      string
	Message   string
	Retryable bool
}

func (e *Error) Error() string { return e.Message }
