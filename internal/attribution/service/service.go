package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/example/adnova/internal/attribution/domain"
	"github.com/example/adnova/internal/attribution/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct{ repo *repository.Repository }

type campaignTotals struct {
	channelInstalls int64
	mmpInstalls     int64
	mmpRevenue      decimal.Decimal
	gameRevenue     decimal.Decimal
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

func DifferenceRate(a, b decimal.Decimal) decimal.Decimal {
	maximum := decimal.Max(a.Abs(), b.Abs())
	if maximum.IsZero() {
		return decimal.Zero
	}
	return a.Sub(b).Abs().Div(maximum).Round(8)
}

func (s *Service) Analyze(ctx context.Context, tenantID, gameID string) (int, error) {
	ads, err := s.repo.LoadAds(ctx, tenantID, gameID)
	if err != nil {
		return 0, err
	}
	mmp, err := s.repo.LoadMMP(ctx, tenantID, gameID)
	if err != nil {
		return 0, err
	}
	revenues, err := s.repo.LoadRevenue(ctx, tenantID, gameID)
	if err != nil {
		return 0, err
	}
	campaigns, err := s.repo.LoadCampaigns(ctx, tenantID, gameID)
	if err != nil {
		return 0, err
	}
	rules, err := s.repo.LoadRules(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	names := map[string]string{}
	totals := map[string]*campaignTotals{}
	for _, campaign := range campaigns {
		names[campaign.ID] = campaign.Name
		totals[campaign.ID] = &campaignTotals{}
	}
	for _, row := range ads {
		ensureTotal(totals, row.CampaignID).channelInstalls += row.Installs
	}
	for _, row := range mmp {
		total := ensureTotal(totals, row.CampaignID)
		total.mmpInstalls += row.Installs
		total.mmpRevenue = total.mmpRevenue.Add(row.Revenue)
	}
	for _, row := range revenues {
		total := ensureTotal(totals, row.CampaignID)
		total.gameRevenue = total.gameRevenue.Add(row.RevenueD7)
	}
	ruleMap := map[string]struct {
		threshold decimal.Decimal
		severity  string
	}{}
	for _, rule := range rules {
		ruleMap[rule.Code] = struct {
			threshold decimal.Decimal
			severity  string
		}{rule.Threshold, rule.Severity}
	}
	findings := make([]domain.Finding, 0)
	for campaignID, total := range totals {
		name := names[campaignID]
		if rule, ok := ruleMap["INSTALL_ATTRIBUTION_GAP"]; ok {
			difference := DifferenceRate(decimal.NewFromInt(total.channelInstalls), decimal.NewFromInt(total.mmpInstalls))
			if difference.GreaterThan(rule.threshold) {
				findings = append(findings, newFinding(tenantID, gameID, campaignID, name, "INSTALL_ATTRIBUTION_GAP", rule.severity, "安装归因存在偏差", fmt.Sprintf("渠道安装数 %d 与 MMP 安装数 %d 的偏差为 %s%%", total.channelInstalls, total.mmpInstalls, difference.Mul(decimal.NewFromInt(100)).StringFixed(1)), difference, map[string]any{"channel_installs": total.channelInstalls, "mmp_installs": total.mmpInstalls, "threshold": rule.threshold}))
			}
		}
		if rule, ok := ruleMap["REVENUE_ATTRIBUTION_GAP"]; ok {
			difference := DifferenceRate(total.mmpRevenue, total.gameRevenue)
			if difference.GreaterThan(rule.threshold) {
				findings = append(findings, newFinding(tenantID, gameID, campaignID, name, "REVENUE_ATTRIBUTION_GAP", rule.severity, "收入归因存在偏差", fmt.Sprintf("MMP 收入 %s 与游戏收入 %s 的偏差为 %s%%", total.mmpRevenue.StringFixed(2), total.gameRevenue.StringFixed(2), difference.Mul(decimal.NewFromInt(100)).StringFixed(1)), difference, map[string]any{"mmp_revenue": total.mmpRevenue, "game_revenue_d7": total.gameRevenue, "threshold": rule.threshold}))
			}
		}
		if rule, ok := ruleMap["DATA_MISSING_OR_DELAY"]; ok && total.channelInstalls > 0 && total.mmpInstalls == 0 {
			findings = append(findings, newFinding(tenantID, gameID, campaignID, name, "DATA_MISSING_OR_DELAY", rule.severity, "MMP 数据缺失或延迟", "渠道侧已有安装，但当前分析周期没有对应的 MMP 数据", decimal.NewFromInt(1), map[string]any{"channel_installs": total.channelInstalls, "mmp_installs": 0}))
		}
	}
	if err := s.repo.Replace(ctx, tenantID, gameID, findings); err != nil {
		return 0, err
	}
	return len(findings), nil
}

func (s *Service) List(ctx context.Context, tenantID, gameID string) ([]domain.Finding, error) {
	return s.repo.List(ctx, tenantID, gameID)
}

func (s *Service) Get(ctx context.Context, tenantID, id string) (*domain.Finding, error) {
	return s.repo.Get(ctx, tenantID, id)
}

func ensureTotal(totals map[string]*campaignTotals, id string) *campaignTotals {
	if totals[id] == nil {
		totals[id] = &campaignTotals{}
	}
	return totals[id]
}

func newFinding(tenantID, gameID, campaignID, campaignName, code, severity, title, description string, difference decimal.Decimal, evidence map[string]any) domain.Finding {
	payload, _ := json.Marshal(evidence)
	return domain.Finding{ID: uuid.NewString(), TenantID: tenantID, GameID: gameID, CampaignID: campaignID, CampaignName: campaignName, RuleCode: code, Severity: severity, Title: title, Description: description, DifferenceRate: difference, Evidence: payload}
}
