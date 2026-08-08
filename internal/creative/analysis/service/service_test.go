package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestFatigueScore(t *testing.T) {
	first := decimal.RequireFromString("0.04")
	last := decimal.RequireFromString("0.01")
	got := FatigueScore(first, last, decimal.RequireFromString("5"))
	if !got.Equal(decimal.RequireFromString("0.85")) {
		t.Fatalf("got %s, want 0.85", got)
	}
}

func TestFatigueScoreCapsFrequency(t *testing.T) {
	got := FatigueScore(decimal.RequireFromString("0.02"), decimal.RequireFromString("0.02"), decimal.NewFromInt(20))
	if !got.Equal(decimal.RequireFromString("0.4")) {
		t.Fatalf("got %s, want 0.4", got)
	}
}

func TestCTRZeroImpressions(t *testing.T) {
	if !CTR(10, 0).IsZero() {
		t.Fatal("CTR should be zero when impressions are zero")
	}
}
