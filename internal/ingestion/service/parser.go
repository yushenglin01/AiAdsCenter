package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/example/adnova/internal/ingestion/domain"
	"github.com/example/adnova/internal/ingestion/repository"
	"github.com/example/adnova/pkg/provider"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type parsedRow struct {
	date          time.Time
	campaignID    string
	creativeID    string
	country       string
	currency      string
	spend         decimal.Decimal
	revenue       decimal.Decimal
	revenueD1     decimal.Decimal
	revenueD3     decimal.Decimal
	revenueD7     decimal.Decimal
	frequency     decimal.Decimal
	impressions   int64
	clicks        int64
	installs      int64
	activations   int64
	registrations int64
	activeUsers   int64
	payers        int64
}

var (
	maxMetricDecimal    = decimal.RequireFromString("99999999999999.999999")
	maxFrequencyDecimal = decimal.RequireFromString("999999.999999")
)

func (s *Service) parseRows(ctx context.Context, input ImportInput, records []provider.Record) ([]parsedRow, time.Time, error) {
	rows := make([]parsedRow, 0, len(records))
	var periodStart time.Time
	campaigns := map[string]string{}
	creatives := map[string]string{}
	for index, record := range records {
		row, err := parseCommon(record)
		if err != nil {
			return nil, periodStart, fmt.Errorf("第 %d 行：%w", index+1, err)
		}
		externalCampaign := record["campaign_external_id"]
		if unsafeText(externalCampaign) {
			return nil, periodStart, fmt.Errorf("第 %d 行：campaign_external_id 包含不安全公式前缀", index+1)
		}
		campaignID, ok := campaigns[externalCampaign]
		if !ok {
			campaign, findErr := s.repo.FindCampaign(ctx, input.TenantID, externalCampaign)
			if findErr != nil || campaign.GameID != input.GameID {
				return nil, periodStart, fmt.Errorf("第 %d 行：广告计划 %q 不存在或不属于该游戏", index+1, externalCampaign)
			}
			campaignID = campaign.ID
			campaigns[externalCampaign] = campaignID
		}
		row.campaignID = campaignID
		if input.ImportType == ImportCreative {
			externalCreative := record["creative_external_id"]
			if unsafeText(externalCreative) {
				return nil, periodStart, fmt.Errorf("第 %d 行：creative_external_id 包含不安全公式前缀", index+1)
			}
			creativeID, ok := creatives[externalCreative]
			if !ok {
				creative, findErr := s.repo.FindCreative(ctx, input.TenantID, externalCreative)
				if findErr != nil || creative.CampaignID != campaignID {
					return nil, periodStart, fmt.Errorf("第 %d 行：素材 %q 不存在或不属于该计划", index+1, externalCreative)
				}
				creativeID = creative.ID
				creatives[externalCreative] = creativeID
			}
			row.creativeID = creativeID
		}
		if err := parseByType(input.ImportType, record, &row); err != nil {
			return nil, periodStart, fmt.Errorf("第 %d 行：%w", index+1, err)
		}
		if periodStart.IsZero() || row.date.Before(periodStart) {
			periodStart = row.date
		}
		rows = append(rows, row)
	}
	return rows, periodStart, nil
}

func parseCommon(record provider.Record) (parsedRow, error) {
	date, err := time.Parse("2006-01-02", record["date"])
	if err != nil {
		return parsedRow{}, fmt.Errorf("date 必须为 YYYY-MM-DD")
	}
	country := strings.ToUpper(record["country"])
	if country == "" {
		country = "ZZ"
	}
	if len(country) != 2 || unsafeText(country) {
		return parsedRow{}, fmt.Errorf("country 必须是两位代码")
	}
	currency := strings.ToUpper(record["currency"])
	if len(currency) != 3 || unsafeText(currency) {
		return parsedRow{}, fmt.Errorf("currency 必须是三位代码")
	}
	if strings.TrimSpace(record["campaign_external_id"]) == "" {
		return parsedRow{}, fmt.Errorf("campaign_external_id 不能为空")
	}
	return parsedRow{date: date.UTC(), country: country, currency: currency}, nil
}

func parseByType(importType string, record provider.Record, row *parsedRow) error {
	var err error
	switch importType {
	case ImportAd:
		if row.spend, err = parseDecimal(record, "spend"); err != nil {
			return err
		}
		if row.impressions, err = parseInt(record, "impressions"); err != nil {
			return err
		}
		if row.clicks, err = parseInt(record, "clicks"); err != nil {
			return err
		}
		if row.installs, err = parseInt(record, "installs"); err != nil {
			return err
		}
	case ImportMMP:
		if row.installs, err = parseInt(record, "installs"); err != nil {
			return err
		}
		if row.activations, err = parseInt(record, "activations"); err != nil {
			return err
		}
		if row.payers, err = parseInt(record, "payers"); err != nil {
			return err
		}
		if row.revenue, err = parseDecimal(record, "revenue"); err != nil {
			return err
		}
	case ImportRevenue:
		if row.registrations, err = parseInt(record, "registrations"); err != nil {
			return err
		}
		if row.activeUsers, err = parseInt(record, "active_users"); err != nil {
			return err
		}
		if row.payers, err = parseInt(record, "payers"); err != nil {
			return err
		}
		if row.revenueD1, err = parseDecimal(record, "revenue_d1"); err != nil {
			return err
		}
		if row.revenueD3, err = parseDecimal(record, "revenue_d3"); err != nil {
			return err
		}
		if row.revenueD7, err = parseDecimal(record, "revenue_d7"); err != nil {
			return err
		}
	case ImportCreative:
		if strings.TrimSpace(record["creative_external_id"]) == "" {
			return fmt.Errorf("creative_external_id 不能为空")
		}
		if row.spend, err = parseDecimal(record, "spend"); err != nil {
			return err
		}
		if row.impressions, err = parseInt(record, "impressions"); err != nil {
			return err
		}
		if row.clicks, err = parseInt(record, "clicks"); err != nil {
			return err
		}
		if row.installs, err = parseInt(record, "installs"); err != nil {
			return err
		}
		if row.frequency, err = parseDecimal(record, "frequency"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的导入类型")
	}
	return nil
}

func parseInt(record provider.Record, key string) (int64, error) {
	value, err := strconv.ParseInt(record[key], 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s 必须是非负整数", key)
	}
	return value, nil
}
func parseDecimal(record provider.Record, key string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(record[key])
	maximum, sqlType := maxMetricDecimal, "DECIMAL(20,6)"
	if key == "frequency" {
		maximum, sqlType = maxFrequencyDecimal, "DECIMAL(12,6)"
	}
	if err != nil || value.IsNegative() || value.Exponent() < -6 || value.GreaterThan(maximum) {
		return decimal.Zero, fmt.Errorf("%s 必须是 %s 范围内的非负数", key, sqlType)
	}
	return value, nil
}
func unsafeText(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && strings.ContainsAny(trimmed[:1], "=+-@")
}

func (s *Service) insert(ctx context.Context, input ImportInput, jobID string, rows []parsedRow) (int, error) {
	switch input.ImportType {
	case ImportAd:
		raw := make([]domain.RawAdMetric, 0, len(rows))
		normalized := make([]domain.NormalizedAdMetric, 0, len(rows))
		for _, row := range rows {
			item := domain.RawAdMetric{ID: uuid.NewString(), TenantID: input.TenantID, ImportJobID: jobID, Source: input.Source, GameID: input.GameID, CampaignID: row.campaignID, Date: row.date, Country: row.country, Currency: row.currency, Spend: row.spend, Impressions: row.impressions, Clicks: row.clicks, Installs: row.installs}
			raw = append(raw, item)
			normalized = append(normalized, domain.NormalizedAdMetric(item))
		}
		return s.repo.InsertAd(ctx, raw, normalized)
	case ImportMMP:
		items := make([]domain.MMPMetric, 0, len(rows))
		for _, row := range rows {
			items = append(items, domain.MMPMetric{ID: uuid.NewString(), TenantID: input.TenantID, ImportJobID: jobID, Source: input.Source, GameID: input.GameID, CampaignID: row.campaignID, Date: row.date, Country: row.country, Installs: row.installs, Activations: row.activations, Payers: row.payers, Revenue: row.revenue, Currency: row.currency})
		}
		if input.Authoritative {
			if input.PeriodEnd.IsZero() || input.PeriodEnd.Before(input.PeriodStart) {
				return 0, fmt.Errorf("权威 MMP 导入日期范围无效")
			}
			return s.repo.ReplaceMMP(ctx, input.TenantID, input.Source, input.GameID, input.PeriodStart, input.PeriodEnd, items)
		}
		return s.repo.InsertMMP(ctx, items)
	case ImportRevenue:
		items := make([]domain.GameRevenueMetric, 0, len(rows))
		for _, row := range rows {
			items = append(items, domain.GameRevenueMetric{ID: uuid.NewString(), TenantID: input.TenantID, ImportJobID: jobID, GameID: input.GameID, CampaignID: row.campaignID, Date: row.date, Country: row.country, Registrations: row.registrations, ActiveUsers: row.activeUsers, Payers: row.payers, RevenueD1: row.revenueD1, RevenueD3: row.revenueD3, RevenueD7: row.revenueD7, Currency: row.currency})
		}
		return s.repo.InsertRevenue(ctx, items)
	case ImportCreative:
		items := make([]domain.CreativeDailyMetric, 0, len(rows))
		for _, row := range rows {
			items = append(items, domain.CreativeDailyMetric{ID: uuid.NewString(), TenantID: input.TenantID, ImportJobID: jobID, GameID: input.GameID, CampaignID: row.campaignID, CreativeID: row.creativeID, Date: row.date, Spend: row.spend, Impressions: row.impressions, Clicks: row.clicks, Installs: row.installs, Frequency: row.frequency, Currency: row.currency})
		}
		return s.repo.InsertCreative(ctx, items)
	default:
		return 0, repository.ErrNotFound
	}
}
