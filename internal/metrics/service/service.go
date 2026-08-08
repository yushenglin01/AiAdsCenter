package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/example/adnova/internal/metrics/domain"
	"github.com/example/adnova/internal/metrics/dto"
	"github.com/example/adnova/internal/metrics/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct{ repo *repository.Repository }

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Recalculate(ctx context.Context, tenantID, gameID string) (int, error) {
	ads, err := s.repo.LoadAds(ctx, tenantID, gameID)
	if err != nil {
		return 0, fmt.Errorf("load normalized ads: %w", err)
	}
	revenues, err := s.repo.LoadRevenue(ctx, tenantID, gameID)
	if err != nil {
		return 0, fmt.Errorf("load game revenue: %w", err)
	}
	campaigns, err := s.repo.LoadCampaigns(ctx, tenantID, gameID)
	if err != nil {
		return 0, fmt.Errorf("load campaigns: %w", err)
	}
	campaignMap := map[string]struct {
		budget   decimal.Decimal
		currency string
	}{}
	for _, c := range campaigns {
		campaignMap[c.ID] = struct {
			budget   decimal.Decimal
			currency string
		}{c.DailyBudget, c.Currency}
	}
	revenueMap := map[string]struct {
		registrations, active, payers int64
		d1, d3, d7                    decimal.Decimal
	}{}
	for _, r := range revenues {
		key := dailyKey(r.CampaignID, r.Date, r.Country)
		revenueMap[key] = struct {
			registrations, active, payers int64
			d1, d3, d7                    decimal.Decimal
		}{r.Registrations, r.ActiveUsers, r.Payers, r.RevenueD1, r.RevenueD3, r.RevenueD7}
	}
	grouped := map[string]*domain.CampaignDailyMetric{}
	for _, ad := range ads {
		key := dailyKey(ad.CampaignID, ad.Date, ad.Country)
		row := grouped[key]
		if row == nil {
			campaign := campaignMap[ad.CampaignID]
			row = &domain.CampaignDailyMetric{ID: uuid.NewString(), TenantID: tenantID, GameID: gameID, CampaignID: ad.CampaignID, Date: ad.Date, Country: ad.Country, Currency: campaign.currency, DailyBudget: campaign.budget}
			grouped[key] = row
		}
		row.Spend = row.Spend.Add(ad.Spend)
		row.Impressions += ad.Impressions
		row.Clicks += ad.Clicks
		row.Installs += ad.Installs
	}
	rows := make([]domain.CampaignDailyMetric, 0, len(grouped))
	for key, row := range grouped {
		rev := revenueMap[key]
		row.Registrations, row.ActiveUsers, row.Payers = rev.registrations, rev.active, rev.payers
		row.RevenueD1, row.RevenueD3, row.RevenueD7 = rev.d1, rev.d3, rev.d7
		applyValues(row, Calculate(toInput(row)))
		rows = append(rows, *row)
	}
	if err := s.repo.Replace(ctx, tenantID, gameID, rows); err != nil {
		return 0, fmt.Errorf("replace campaign metrics: %w", err)
	}
	return len(rows), nil
}

func (s *Service) Campaigns(ctx context.Context, tenantID, gameID string) ([]dto.CampaignMetric, error) {
	daily, err := s.repo.ListDaily(ctx, tenantID, gameID, "")
	if err != nil {
		return nil, err
	}
	campaigns, err := s.repo.LoadCampaigns(ctx, tenantID, gameID)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	countries := map[string]string{}
	for _, c := range campaigns {
		names[c.ID] = c.Name
		countries[c.ID] = c.Country
	}
	grouped := map[string]*dto.CampaignMetric{}
	for _, row := range daily {
		metric := grouped[row.CampaignID]
		if metric == nil {
			metric = &dto.CampaignMetric{CampaignID: row.CampaignID, CampaignName: names[row.CampaignID], Country: countries[row.CampaignID], Currency: row.Currency}
			grouped[row.CampaignID] = metric
		}
		addDaily(metric, row)
	}
	result := make([]dto.CampaignMetric, 0, len(grouped))
	for _, row := range grouped {
		applyAggregate(row)
		result = append(result, *row)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Spend.GreaterThan(result[j].Spend) })
	return result, nil
}
func (s *Service) Campaign(ctx context.Context, tenantID, campaignID string) (*dto.CampaignMetric, error) {
	rows, err := s.Campaigns(ctx, tenantID, "")
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].CampaignID == campaignID {
			return &rows[i], nil
		}
	}
	return nil, fmt.Errorf("campaign metrics not found")
}
func (s *Service) Trends(ctx context.Context, tenantID, gameID, campaignID string) ([]dto.TrendPoint, error) {
	daily, err := s.repo.ListDaily(ctx, tenantID, gameID, campaignID)
	if err != nil {
		return nil, err
	}
	grouped := map[string]*dto.CampaignMetric{}
	dates := map[string]time.Time{}
	for _, row := range daily {
		key := row.Date.Format("2006-01-02")
		metric := grouped[key]
		if metric == nil {
			metric = &dto.CampaignMetric{Currency: row.Currency}
			grouped[key] = metric
			dates[key] = row.Date
		}
		addDaily(metric, row)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]dto.TrendPoint, 0, len(keys))
	for _, key := range keys {
		metric := grouped[key]
		applyAggregate(metric)
		result = append(result, dto.TrendPoint{Date: dates[key], Spend: metric.Spend, RevenueD1: metric.RevenueD1, RevenueD7: metric.RevenueD7, CPI: metric.CPI, ROASD1: metric.ROASD1, ROASD7: metric.ROASD7})
	}
	return result, nil
}

func (s *Service) Daily(ctx context.Context, tenantID, gameID string) ([]domain.CampaignDailyMetric, error) {
	return s.repo.ListDaily(ctx, tenantID, gameID, "")
}
func (s *Service) Overview(ctx context.Context, tenantID, gameID string) (*dto.Overview, error) {
	campaigns, err := s.Campaigns(ctx, tenantID, gameID)
	if err != nil {
		return nil, err
	}
	overview := &dto.Overview{}
	for _, row := range campaigns {
		mergeMetric(&overview.CampaignMetric, row)
	}
	applyAggregate(&overview.CampaignMetric)
	business, attribution, creative, err := s.repo.RiskCounts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	overview.HighRiskCampaigns, overview.AttributionAnomalies, overview.FatiguedCreatives = business, attribution, creative
	return overview, nil
}

func dailyKey(campaignID string, date time.Time, country string) string {
	return campaignID + "|" + date.Format("2006-01-02") + "|" + country
}
func toInput(row *domain.CampaignDailyMetric) domain.MetricInput {
	return domain.MetricInput{Spend: row.Spend, DailyBudget: row.DailyBudget, RevenueD1: row.RevenueD1, RevenueD3: row.RevenueD3, RevenueD7: row.RevenueD7, Impressions: row.Impressions, Clicks: row.Clicks, Installs: row.Installs, Registrations: row.Registrations, ActiveUsers: row.ActiveUsers, Payers: row.Payers}
}
func applyValues(row *domain.CampaignDailyMetric, v domain.MetricValues) {
	row.CTR, row.CVR, row.CPI, row.CPA, row.PayerRate = v.CTR, v.CVR, v.CPI, v.CPA, v.PayerRate
	row.ROASD1, row.ROASD3, row.ROASD7, row.BudgetConsumption, row.LTVD7 = v.ROASD1, v.ROASD3, v.ROASD7, v.BudgetConsumption, v.LTVD7
}
func addDaily(target *dto.CampaignMetric, row domain.CampaignDailyMetric) {
	target.Spend = target.Spend.Add(row.Spend)
	target.DailyBudget = target.DailyBudget.Add(row.DailyBudget)
	target.RevenueD1 = target.RevenueD1.Add(row.RevenueD1)
	target.RevenueD3 = target.RevenueD3.Add(row.RevenueD3)
	target.RevenueD7 = target.RevenueD7.Add(row.RevenueD7)
	target.Impressions += row.Impressions
	target.Clicks += row.Clicks
	target.Installs += row.Installs
	target.Registrations += row.Registrations
	target.ActiveUsers += row.ActiveUsers
	target.Payers += row.Payers
}
func mergeMetric(target *dto.CampaignMetric, row dto.CampaignMetric) {
	target.Currency = row.Currency
	target.Spend = target.Spend.Add(row.Spend)
	target.DailyBudget = target.DailyBudget.Add(row.DailyBudget)
	target.RevenueD1 = target.RevenueD1.Add(row.RevenueD1)
	target.RevenueD3 = target.RevenueD3.Add(row.RevenueD3)
	target.RevenueD7 = target.RevenueD7.Add(row.RevenueD7)
	target.Impressions += row.Impressions
	target.Clicks += row.Clicks
	target.Installs += row.Installs
	target.Registrations += row.Registrations
	target.ActiveUsers += row.ActiveUsers
	target.Payers += row.Payers
}
func applyAggregate(row *dto.CampaignMetric) {
	values := Calculate(domain.MetricInput{Spend: row.Spend, DailyBudget: row.DailyBudget, RevenueD1: row.RevenueD1, RevenueD3: row.RevenueD3, RevenueD7: row.RevenueD7, Impressions: row.Impressions, Clicks: row.Clicks, Installs: row.Installs, Registrations: row.Registrations, ActiveUsers: row.ActiveUsers, Payers: row.Payers})
	row.CTR, row.CVR, row.CPI, row.CPA, row.PayerRate = values.CTR, values.CVR, values.CPI, values.CPA, values.PayerRate
	row.ROASD1, row.ROASD3, row.ROASD7, row.BudgetConsumption, row.LTVD7 = values.ROASD1, values.ROASD3, values.ROASD7, values.BudgetConsumption, values.LTVD7
}
