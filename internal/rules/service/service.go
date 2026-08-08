package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	metricsdomain "github.com/example/adnova/internal/metrics/domain"
	metricsdto "github.com/example/adnova/internal/metrics/dto"
	metricsservice "github.com/example/adnova/internal/metrics/service"
	"github.com/example/adnova/internal/rules/domain"
	"github.com/example/adnova/internal/rules/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct {
	repo    *repository.Repository
	metrics *metricsservice.Service
}

type UpdateRuleInput struct {
	Threshold       *decimal.Decimal `json:"threshold"`
	ConsecutiveDays *int             `json:"consecutive_days"`
	Enabled         *bool            `json:"enabled"`
}

type RuleOverview struct {
	Rules      []domain.AnalysisRule      `json:"rules"`
	Benchmarks []domain.BusinessBenchmark `json:"benchmarks"`
	Findings   []domain.RuleFinding       `json:"findings"`
}

func New(repo *repository.Repository, metrics *metricsservice.Service) *Service {
	return &Service{repo: repo, metrics: metrics}
}

func (s *Service) List(ctx context.Context, tenantID, gameID string) (*RuleOverview, error) {
	rules, err := s.repo.ListRules(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	findings, err := s.repo.ListFindings(ctx, tenantID, gameID)
	if err != nil {
		return nil, err
	}
	benchmarks, err := s.repo.ListBenchmarks(ctx, tenantID, gameID)
	if err != nil {
		return nil, err
	}
	return &RuleOverview{Rules: rules, Benchmarks: benchmarks, Findings: findings}, nil
}

func (s *Service) Update(ctx context.Context, tenantID, id string, input UpdateRuleInput) (*domain.AnalysisRule, error) {
	values := map[string]any{}
	if input.Threshold != nil {
		values["threshold"] = *input.Threshold
	}
	if input.ConsecutiveDays != nil {
		if *input.ConsecutiveDays < 1 || *input.ConsecutiveDays > 30 {
			return nil, fmt.Errorf("consecutive_days must be between 1 and 30")
		}
		values["consecutive_days"] = *input.ConsecutiveDays
	}
	if input.Enabled != nil {
		values["enabled"] = *input.Enabled
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	return s.repo.UpdateRule(ctx, tenantID, id, values)
}

func (s *Service) Analyze(ctx context.Context, tenantID, gameID string) (int, error) {
	rules, err := s.repo.ListRules(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	campaigns, err := s.metrics.Campaigns(ctx, tenantID, gameID)
	if err != nil {
		return 0, err
	}
	byCode := make(map[string]domain.AnalysisRule, len(rules))
	for _, rule := range rules {
		if rule.Enabled {
			byCode[rule.Code] = rule
		}
	}
	benchmarks, err := s.repo.ListBenchmarks(ctx, tenantID, gameID)
	if err != nil {
		return 0, err
	}
	for _, benchmark := range benchmarks {
		code := map[string]string{"TARGET_ROAS_D7": "ROAS_BELOW_TARGET", "BENCHMARK_CPI": "CPI_ABOVE_BENCHMARK", "BENCHMARK_PAYER_RATE": "PAYER_RATE_BELOW_BENCHMARK"}[benchmark.MetricCode]
		if rule, ok := byCode[code]; ok {
			rule.Threshold = benchmark.Value
			byCode[code] = rule
		}
	}
	findings := make([]domain.RuleFinding, 0)
	for _, campaign := range campaigns {
		findings = append(findings, evaluateCampaign(tenantID, gameID, campaign, byCode)...)
	}
	daily, err := s.metrics.Daily(ctx, tenantID, gameID)
	if err != nil {
		return 0, err
	}
	if rule, ok := byCode["D1_ROAS_DECLINE"]; ok {
		grouped := map[string][]metricsdomain.CampaignDailyMetric{}
		for _, row := range daily {
			grouped[row.CampaignID] = append(grouped[row.CampaignID], row)
		}
		for campaignID, rows := range grouped {
			if declined, values := consecutiveD1Decline(rows, rule.ConsecutiveDays); declined {
				payload, _ := json.Marshal(map[string]any{"roas_d1": values, "consecutive_days": rule.ConsecutiveDays})
				findings = append(findings, domain.RuleFinding{ID: uuid.NewString(), TenantID: tenantID, GameID: gameID, CampaignID: campaignID, RuleCode: rule.Code, Severity: rule.Severity, Title: "D1 ROAS 连续下降", Description: fmt.Sprintf("最近 %d 天 D1 ROAS 连续下降，需检查流量质量变化", rule.ConsecutiveDays), Evidence: payload})
			}
		}
	}
	if err := s.repo.ReplaceFindings(ctx, tenantID, gameID, findings); err != nil {
		return 0, err
	}
	return len(findings), nil
}

func consecutiveD1Decline(rows []metricsdomain.CampaignDailyMetric, days int) (bool, []decimal.Decimal) {
	if days < 2 || len(rows) < days {
		return false, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Date.Before(rows[j].Date) })
	rows = rows[len(rows)-days:]
	values := make([]decimal.Decimal, len(rows))
	for i, row := range rows {
		values[i] = row.ROASD1
		if i > 0 && !values[i].LessThan(values[i-1]) {
			return false, values
		}
	}
	return true, values
}

func evaluateCampaign(tenantID, gameID string, metric metricsdto.CampaignMetric, rules map[string]domain.AnalysisRule) []domain.RuleFinding {
	result := make([]domain.RuleFinding, 0, 4)
	check := func(code string, triggered bool, title, description string, evidence map[string]any) {
		rule, ok := rules[code]
		if !ok || !triggered {
			return
		}
		payload, _ := json.Marshal(evidence)
		result = append(result, domain.RuleFinding{ID: uuid.NewString(), TenantID: tenantID, GameID: gameID, CampaignID: metric.CampaignID, RuleCode: code, Severity: rule.Severity, Title: title, Description: description, Evidence: payload})
	}
	if rule, ok := rules["ROAS_BELOW_TARGET"]; ok {
		check(rule.Code, metric.ROASD7.LessThan(rule.Threshold), "D7 ROAS 低于目标", fmt.Sprintf("%s 的 D7 ROAS 为 %s，低于目标 %s", metric.CampaignName, metric.ROASD7.StringFixed(2), rule.Threshold.StringFixed(2)), map[string]any{"actual": metric.ROASD7, "threshold": rule.Threshold, "metric": "roas_d7"})
	}
	if rule, ok := rules["CPI_ABOVE_BENCHMARK"]; ok {
		check(rule.Code, metric.CPI.GreaterThan(rule.Threshold), "获客成本高于基准", fmt.Sprintf("%s 的 CPI 为 %s，高于基准 %s", metric.CampaignName, metric.CPI.StringFixed(2), rule.Threshold.StringFixed(2)), map[string]any{"actual": metric.CPI, "threshold": rule.Threshold, "metric": "cpi"})
	}
	if rule, ok := rules["PAYER_RATE_BELOW_BENCHMARK"]; ok {
		check(rule.Code, metric.PayerRate.LessThan(rule.Threshold), "付费率低于基准", fmt.Sprintf("%s 的付费率为 %s%%，低于基准 %s%%", metric.CampaignName, metric.PayerRate.Mul(decimal.NewFromInt(100)).StringFixed(2), rule.Threshold.Mul(decimal.NewFromInt(100)).StringFixed(2)), map[string]any{"actual": metric.PayerRate, "threshold": rule.Threshold, "metric": "payer_rate"})
	}
	if rule, ok := rules["BUDGET_OVER_90_ROAS_LOW"]; ok {
		target := decimal.RequireFromString("1.30")
		check(rule.Code, metric.BudgetConsumption.GreaterThanOrEqual(rule.Threshold) && metric.ROASD7.LessThan(target), "预算消耗偏高且回收不足", fmt.Sprintf("%s 已消耗预算的 %s%%，D7 ROAS 仅 %s", metric.CampaignName, metric.BudgetConsumption.Mul(decimal.NewFromInt(100)).StringFixed(1), metric.ROASD7.StringFixed(2)), map[string]any{"budget_consumption_rate": metric.BudgetConsumption, "roas_d7": metric.ROASD7, "target_roas": target})
	}
	return result
}
