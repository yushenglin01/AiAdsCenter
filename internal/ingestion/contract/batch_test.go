package contract

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validBatch() Batch {
	return Batch{
		Schema: SchemaName, SchemaVersion: SchemaVersion, BatchID: "partner-20260805-001",
		DatasetType: DatasetMMP, Source: "appsflyer", Producer: Producer{System: "partner-data-hub"},
		GameCode: "game-1001", Timezone: "UTC", Period: Period{Start: "2026-08-01", End: "2026-08-02"},
		CollectedAt: "2026-08-02T10:30:00Z",
		Records: []map[string]any{{
			"date": "2026-08-01", "campaign_external_id": "meta-us-001", "currency": "USD",
			"installs": 10, "activations": 8, "payers": 2, "revenue": "12.50",
		}},
	}
}

func TestBatchNormalizeAndValidate(t *testing.T) {
	batch := validBatch()
	batch.Normalize()
	require.NoError(t, batch.Validate())
	require.Equal(t, "APPSFLYER", batch.Source)
	require.Equal(t, "PARTNER-DATA-HUB", batch.Producer.System)
	require.Equal(t, WriteAppend, batch.WriteMode)
}

func TestBatchRejectsRecordOutsidePeriod(t *testing.T) {
	batch := validBatch()
	batch.Records[0]["date"] = "2026-08-03"
	batch.Normalize()
	require.ErrorContains(t, batch.Validate(), "不在 period 范围内")
}

func TestBatchRestrictsReplaceRangeToMMP(t *testing.T) {
	batch := validBatch()
	batch.DatasetType = DatasetAd
	batch.WriteMode = WriteReplaceRange
	batch.Normalize()
	require.ErrorContains(t, batch.Validate(), "仅允许用于 MMP_METRICS")
}

func TestBatchRequiresUTC(t *testing.T) {
	batch := validBatch()
	batch.Timezone = "Asia/Shanghai"
	batch.Normalize()
	require.ErrorContains(t, batch.Validate(), "仅支持 UTC")
}

func TestDecodeRejectsUnknownEnvelopeField(t *testing.T) {
	_, err := Decode([]byte(`{"schema":"adnova.ingestion.batch","unexpected":true}`))
	require.ErrorContains(t, err, "unknown field")
}

func TestBatchRejectsUnknownRecordField(t *testing.T) {
	batch := validBatch()
	batch.Records[0]["unexpected"] = true
	batch.Normalize()
	require.ErrorContains(t, batch.Validate(), "未知字段")
}

func TestBatchRequiresDatasetFields(t *testing.T) {
	batch := validBatch()
	delete(batch.Records[0], "revenue")
	batch.Normalize()
	require.ErrorContains(t, batch.Validate(), "缺少必填字段 revenue")
}
