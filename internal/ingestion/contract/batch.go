package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SchemaName    = "adnova.ingestion.batch"
	SchemaVersion = "1.0.0"

	DatasetAd       = "AD_METRICS"
	DatasetMMP      = "MMP_METRICS"
	DatasetRevenue  = "GAME_REVENUE"
	DatasetCreative = "CREATIVE_METRICS"

	WriteAppend       = "APPEND"
	WriteReplaceRange = "REPLACE_RANGE"
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	sourcePattern     = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,39}$`)
)

var allowedRecordFields = map[string]bool{
	"source_record_id": true, "date": true, "channel_code": true,
	"campaign_external_id": true, "creative_external_id": true,
	"country": true, "currency": true, "spend": true, "revenue": true,
	"revenue_d1": true, "revenue_d3": true, "revenue_d7": true,
	"frequency": true, "impressions": true, "clicks": true, "installs": true,
	"activations": true, "registrations": true, "active_users": true, "payers": true,
}

type Producer struct {
	System   string `json:"system"`
	Instance string `json:"instance,omitempty"`
	Version  string `json:"version,omitempty"`
}

type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Batch struct {
	Schema        string           `json:"schema"`
	SchemaVersion string           `json:"schema_version"`
	BatchID       string           `json:"batch_id"`
	TenantID      string           `json:"tenant_id,omitempty"`
	DatasetType   string           `json:"dataset_type"`
	Source        string           `json:"source"`
	WriteMode     string           `json:"write_mode"`
	Producer      Producer         `json:"producer"`
	GameCode      string           `json:"game_code"`
	Timezone      string           `json:"timezone"`
	Period        Period           `json:"period"`
	CollectedAt   string           `json:"collected_at"`
	Records       []map[string]any `json:"records"`
}

func Decode(data []byte) (Batch, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var batch Batch
	if err := decoder.Decode(&batch); err != nil {
		return Batch{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Batch{}, fmt.Errorf("JSON 只能包含一个批次对象")
		}
		return Batch{}, err
	}
	return batch, nil
}

func (b *Batch) Normalize() {
	b.Schema = strings.TrimSpace(b.Schema)
	b.SchemaVersion = strings.TrimSpace(b.SchemaVersion)
	b.BatchID = strings.TrimSpace(b.BatchID)
	b.TenantID = strings.TrimSpace(b.TenantID)
	b.DatasetType = strings.ToUpper(strings.TrimSpace(b.DatasetType))
	b.Source = strings.ToUpper(strings.TrimSpace(b.Source))
	b.WriteMode = strings.ToUpper(strings.TrimSpace(b.WriteMode))
	b.Producer.System = strings.ToUpper(strings.TrimSpace(b.Producer.System))
	b.Producer.Instance = strings.TrimSpace(b.Producer.Instance)
	b.Producer.Version = strings.TrimSpace(b.Producer.Version)
	b.GameCode = strings.TrimSpace(b.GameCode)
	b.Timezone = strings.TrimSpace(b.Timezone)
	b.Period.Start = strings.TrimSpace(b.Period.Start)
	b.Period.End = strings.TrimSpace(b.Period.End)
	b.CollectedAt = strings.TrimSpace(b.CollectedAt)
	if b.WriteMode == "" {
		b.WriteMode = WriteAppend
	}
	if b.Timezone == "" {
		b.Timezone = "UTC"
	}
}

func (b Batch) Validate() error {
	if b.Schema != SchemaName || b.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema 必须为 %s，schema_version 必须为 %s", SchemaName, SchemaVersion)
	}
	if !identifierPattern.MatchString(b.BatchID) {
		return fmt.Errorf("batch_id 格式无效")
	}
	if b.TenantID != "" {
		if _, err := uuid.Parse(b.TenantID); err != nil {
			return fmt.Errorf("tenant_id 必须为 UUID")
		}
	}
	if !identifierPattern.MatchString(b.Producer.System) || len(b.Producer.System) > 80 {
		return fmt.Errorf("producer.system 格式无效")
	}
	if len(b.Producer.Instance) > 128 || len(b.Producer.Version) > 64 {
		return fmt.Errorf("producer.instance 或 producer.version 过长")
	}
	if !sourcePattern.MatchString(b.Source) {
		return fmt.Errorf("source 格式无效")
	}
	if !identifierPattern.MatchString(b.GameCode) || len(b.GameCode) > 80 {
		return fmt.Errorf("game_code 格式无效")
	}
	if b.Timezone != "UTC" {
		return fmt.Errorf("首版标准输入 timezone 仅支持 UTC")
	}
	if len(b.Records) == 0 || len(b.Records) > 10_000 {
		return fmt.Errorf("records 必须包含 1 至 10000 行")
	}
	start, err := time.Parse("2006-01-02", b.Period.Start)
	if err != nil {
		return fmt.Errorf("period.start 必须为 YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", b.Period.End)
	if err != nil || end.Before(start) {
		return fmt.Errorf("period.end 必须为不早于 start 的 YYYY-MM-DD")
	}
	if _, err := time.Parse(time.RFC3339, b.CollectedAt); err != nil {
		return fmt.Errorf("collected_at 必须为 RFC3339 时间")
	}
	allowedDatasets := map[string]bool{DatasetAd: true, DatasetMMP: true, DatasetRevenue: true, DatasetCreative: true}
	if !allowedDatasets[b.DatasetType] {
		return fmt.Errorf("dataset_type 不受支持")
	}
	if b.WriteMode != WriteAppend && b.WriteMode != WriteReplaceRange {
		return fmt.Errorf("write_mode 必须为 APPEND 或 REPLACE_RANGE")
	}
	if b.WriteMode == WriteReplaceRange && b.DatasetType != DatasetMMP {
		return fmt.Errorf("REPLACE_RANGE 仅允许用于 MMP_METRICS")
	}
	for index, record := range b.Records {
		for key := range record {
			if !allowedRecordFields[key] {
				return fmt.Errorf("第 %d 行包含未知字段 %s", index+1, key)
			}
		}
		dateValue, ok := record["date"].(string)
		if !ok {
			return fmt.Errorf("第 %d 行 date 必须是字符串", index+1)
		}
		date, err := time.Parse("2006-01-02", dateValue)
		if err != nil || date.Before(start) || date.After(end) {
			return fmt.Errorf("第 %d 行 date 不在 period 范围内", index+1)
		}
		if err := validateRecordStrings(record); err != nil {
			return fmt.Errorf("第 %d 行：%w", index+1, err)
		}
		if err := validateRequiredFields(b.DatasetType, record); err != nil {
			return fmt.Errorf("第 %d 行：%w", index+1, err)
		}
	}
	return nil
}

func validateRequiredFields(dataset string, record map[string]any) error {
	required := map[string][]string{
		DatasetAd:       {"spend", "impressions", "clicks", "installs"},
		DatasetMMP:      {"installs", "activations", "payers", "revenue"},
		DatasetRevenue:  {"registrations", "active_users", "payers", "revenue_d1", "revenue_d3", "revenue_d7"},
		DatasetCreative: {"creative_external_id", "spend", "impressions", "clicks", "installs", "frequency"},
	}
	for _, key := range required[dataset] {
		if _, ok := record[key]; !ok {
			return fmt.Errorf("缺少必填字段 %s", key)
		}
	}
	return nil
}

func validateRecordStrings(record map[string]any) error {
	limits := map[string]int{
		"source_record_id": 160, "channel_code": 40, "campaign_external_id": 120,
		"creative_external_id": 120, "country": 2, "currency": 3,
	}
	for key, limit := range limits {
		value, exists := record[key]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok || len(text) > limit {
			return fmt.Errorf("%s 必须是长度不超过 %d 的字符串", key, limit)
		}
	}
	for _, key := range []string{"campaign_external_id", "currency"} {
		if value, ok := record[key].(string); !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s 不能为空", key)
		}
	}
	if country, ok := record["country"].(string); ok && country != "" && (len(country) != 2 || country != strings.ToUpper(country)) {
		return fmt.Errorf("country 必须为两位大写代码")
	}
	if currency, ok := record["currency"].(string); !ok || len(currency) != 3 || currency != strings.ToUpper(currency) {
		return fmt.Errorf("currency 必须为三位大写代码")
	}
	return nil
}

func (b Batch) PeriodTimes() (time.Time, time.Time) {
	start, _ := time.Parse("2006-01-02", b.Period.Start)
	end, _ := time.Parse("2006-01-02", b.Period.End)
	return start.UTC(), end.UTC()
}
