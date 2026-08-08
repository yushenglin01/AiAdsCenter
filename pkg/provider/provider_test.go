package provider

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCSVProvider(t *testing.T) {
	records, err := (CSVProvider{}).Decode(strings.NewReader("date,spend\n2026-07-01,10.250000\n"))
	require.NoError(t, err)
	require.Equal(t, "10.250000", records[0]["spend"])
}

func TestJSONProviderPreservesDecimalText(t *testing.T) {
	records, err := (JSONProvider{}).Decode(strings.NewReader(`[{"spend":10.123456,"installs":0}]`))
	require.NoError(t, err)
	require.Equal(t, "10.123456", records[0]["spend"])
}

func TestProvidersRejectMalformedRows(t *testing.T) {
	_, err := (CSVProvider{}).Decode(strings.NewReader("a,b\n1\n"))
	require.Error(t, err)
	_, err = (JSONProvider{}).Decode(strings.NewReader(`[{"nested":{"bad":true}}]`))
	require.Error(t, err)
}
