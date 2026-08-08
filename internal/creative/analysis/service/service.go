package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	creativerepository "github.com/example/adnova/internal/creative/analysis/repository"
	creativedomain "github.com/example/adnova/internal/creative/domain"
	ingestiondomain "github.com/example/adnova/internal/ingestion/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct {
	repo *creativerepository.Repository
}

type ruleConfig struct {
	threshold decimal.Decimal
	severity  string
}

func New(repo *creativerepository.Repository) *Service { return &Service{repo: repo} }

func CTR(clicks, impressions int64) decimal.Decimal {
	if impressions == 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(clicks).Div(decimal.NewFromInt(impressions)).Round(8)
}

func FatigueScore(firstCTR, lastCTR, frequency decimal.Decimal) decimal.Decimal {
	decline := decimal.Zero
	if firstCTR.GreaterThan(decimal.Zero) && lastCTR.LessThan(firstCTR) {
		decline = firstCTR.Sub(lastCTR).Div(firstCTR)
	}
	frequencyScore := frequency.Div(decimal.NewFromInt(5))
	if frequencyScore.GreaterThan(decimal.NewFromInt(1)) {
		frequencyScore = decimal.NewFromInt(1)
	}
	return decline.Mul(decimal.RequireFromString("0.6")).Add(frequencyScore.Mul(decimal.RequireFromString("0.4"))).Round(8)
}

func (s *Service) Analyze(ctx context.Context, tenantID, gameID string) (int, error) {
	metrics, err := s.repo.LoadMetrics(ctx, tenantID, gameID)
	if err != nil {
		return 0, err
	}
	creatives, err := s.repo.LoadCreatives(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	rules, err := s.repo.LoadRules(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	names := map[string]string{}
	for _, creative := range creatives {
		names[creative.ID] = creative.Name
	}
	config := map[string]ruleConfig{}
	for _, rule := range rules {
		config[rule.Code] = ruleConfig{threshold: rule.Threshold, severity: rule.Severity}
	}
	grouped := map[string][]ingestiondomain.CreativeDailyMetric{}
	for _, row := range metrics {
		grouped[row.CreativeID] = append(grouped[row.CreativeID], row)
	}
	findings := make([]creativedomain.Finding, 0)
	for creativeID, rows := range grouped {
		sort.Slice(rows, func(i, j int) bool { return rows[i].Date.Before(rows[j].Date) })
		if len(rows) > 7 {
			rows = rows[len(rows)-7:]
		}
		first, last := rows[0], rows[len(rows)-1]
		periodSpend := decimal.Zero
		var periodInstalls int64
		for _, row := range rows {
			periodSpend = periodSpend.Add(row.Spend)
			periodInstalls += row.Installs
		}
		firstCTR := CTR(first.Clicks, first.Impressions)
		lastCTR := CTR(last.Clicks, last.Impressions)
		decline := decimal.Zero
		if firstCTR.GreaterThan(decimal.Zero) && lastCTR.LessThan(firstCTR) {
			decline = firstCTR.Sub(lastCTR).Div(firstCTR).Round(8)
		}
		score := FatigueScore(firstCTR, lastCTR, last.Frequency)
		base := creativedomain.Finding{TenantID: tenantID, GameID: gameID, CampaignID: last.CampaignID, CreativeID: creativeID, CreativeName: names[creativeID], FatigueScore: score, CTRChange7D: decline, Frequency: last.Frequency}
		if rule, ok := config["CREATIVE_CTR_DECLINE"]; ok && decline.GreaterThan(rule.threshold) {
			findings = append(findings, makeFinding(base, "CREATIVE_CTR_DECLINE", rule.severity, "素材 CTR 持续下降", fmt.Sprintf("近 7 日 CTR 从 %s%% 降至 %s%%，降幅 %s%%", firstCTR.Mul(decimal.NewFromInt(100)).StringFixed(2), lastCTR.Mul(decimal.NewFromInt(100)).StringFixed(2), decline.Mul(decimal.NewFromInt(100)).StringFixed(1)), map[string]any{"first_ctr": firstCTR, "last_ctr": lastCTR, "decline_rate": decline, "threshold": rule.threshold}))
		}
		if rule, ok := config["CREATIVE_FREQUENCY_HIGH"]; ok && last.Frequency.GreaterThan(rule.threshold) {
			findings = append(findings, makeFinding(base, "CREATIVE_FREQUENCY_HIGH", rule.severity, "素材曝光频次偏高", fmt.Sprintf("最新频次 %s，超过阈值 %s", last.Frequency.StringFixed(2), rule.threshold.StringFixed(2)), map[string]any{"frequency": last.Frequency, "threshold": rule.threshold}))
		}
		if rule, ok := config["CREATIVE_FATIGUE_HIGH"]; ok && score.GreaterThan(rule.threshold) {
			findings = append(findings, makeFinding(base, "CREATIVE_FATIGUE_HIGH", rule.severity, "素材疲劳风险高", fmt.Sprintf("确定性疲劳评分为 %s，建议降低频次并准备替换素材", score.StringFixed(2)), map[string]any{"fatigue_score": score, "ctr_decline": decline, "frequency": last.Frequency, "threshold": rule.threshold}))
		}
		if rule, ok := config["HIGH_SPEND_LOW_CONVERSION"]; ok {
			periodCPI := decimal.Zero
			if periodInstalls > 0 {
				periodCPI = periodSpend.Div(decimal.NewFromInt(periodInstalls)).Round(8)
			}
			if periodInstalls == 0 || periodCPI.GreaterThan(rule.threshold) {
				findings = append(findings, makeFinding(base, "HIGH_SPEND_LOW_CONVERSION", rule.severity, "素材高消耗低转化", fmt.Sprintf("近 7 日消耗 %s，安装 %d，CPI %s 高于阈值 %s", periodSpend.StringFixed(2), periodInstalls, periodCPI.StringFixed(2), rule.threshold.StringFixed(2)), map[string]any{"spend": periodSpend, "installs": periodInstalls, "cpi": periodCPI, "threshold": rule.threshold}))
			}
		}
	}
	if err := s.repo.Replace(ctx, tenantID, gameID, findings); err != nil {
		return 0, err
	}
	return len(findings), nil
}

func (s *Service) List(ctx context.Context, tenantID, gameID string) ([]creativedomain.Finding, error) {
	return s.repo.List(ctx, tenantID, gameID)
}

func (s *Service) Get(ctx context.Context, tenantID, id string) (*creativedomain.Finding, error) {
	return s.repo.Get(ctx, tenantID, id)
}

func makeFinding(base creativedomain.Finding, code, severity, title, description string, evidence map[string]any) creativedomain.Finding {
	payload, _ := json.Marshal(evidence)
	base.ID = uuid.NewString()
	base.RuleCode = code
	base.Severity = severity
	base.Title = title
	base.Description = description
	base.Evidence = payload
	return base
}
