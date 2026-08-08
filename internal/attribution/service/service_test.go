package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestDifferenceRate(t *testing.T) {
	cases := []struct {
		name string
		a, b decimal.Decimal
		want string
	}{
		{"same", decimal.NewFromInt(100), decimal.NewFromInt(100), "0"},
		{"twelve percent", decimal.NewFromInt(100), decimal.NewFromInt(88), "0.12"},
		{"symmetric", decimal.NewFromInt(88), decimal.NewFromInt(100), "0.12"},
		{"both zero", decimal.Zero, decimal.Zero, "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DifferenceRate(tc.a, tc.b)
			if !got.Equal(decimal.RequireFromString(tc.want)) {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}
