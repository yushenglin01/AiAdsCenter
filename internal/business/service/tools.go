package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agenttools "github.com/example/adnova/internal/agent/tools"
	auditdomain "github.com/example/adnova/internal/audit/domain"
	"github.com/example/adnova/internal/business/domain"
	"github.com/example/adnova/internal/common/identity"
	metricsdto "github.com/example/adnova/internal/metrics/dto"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/google/uuid"
)

type readArgs struct {
	GameID       string `json:"game_id"`
	CampaignID   string `json:"campaign_id"`
	AnalysisDate string `json:"analysis_date"`
}

type approvalArgs struct {
	RecommendationID string `json:"recommendation_id"`
	ReportID         string `json:"report_id"`
}

func (s *Service) registerTools() error {
	definitions := []agenttools.Definition{
		{Name: "get_campaign_metrics", Description: "读取指定计划的确定性经营指标", Permission: "READ_METRICS", Handler: s.getCampaignMetrics},
		{Name: "get_business_benchmark", Description: "读取游戏历史经营基准", Permission: "READ_METRICS", Handler: s.getBenchmarks},
		{Name: "get_attribution_anomalies", Description: "读取指定计划归因异常", Permission: "READ_ANALYSIS", Handler: s.getAttribution},
		{Name: "get_creative_findings", Description: "读取指定计划素材风险", Permission: "READ_ANALYSIS", Handler: s.getCreative},
		{Name: "get_verified_research_sources", Description: "读取人工核验且保留来源的政策、竞品和市场资料", Permission: "READ_VERIFIED_RESEARCH", Handler: s.getVerifiedResearch},
		{Name: "forecast_ltv", Description: "返回基于当前 D7 观测值的确定性 LTV 代理", Permission: "READ_METRICS", Handler: s.forecastLTV},
		{Name: "create_recommendation", Description: "创建建议，不执行广告平台变更", Permission: "CREATE_RECOMMENDATION", Handler: s.createRecommendation},
		{Name: "create_approval_request", Description: "为高风险建议创建待审批请求", Permission: "CREATE_APPROVAL_REQUEST", Handler: s.createApproval},
	}
	for _, definition := range definitions {
		if err := s.registry.Register(definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) collectInput(ctx context.Context, tenantID, userID, taskID, gameID, campaignID, analysisDate string) (*BusinessInput, error) {
	toolContext := agenttools.Context{TenantID: tenantID, UserID: userID, TaskID: taskID}
	args, _ := json.Marshal(readArgs{GameID: gameID, CampaignID: campaignID, AnalysisDate: analysisDate})
	execute := func(name string) (json.RawMessage, error) {
		return s.executeTool(ctx, name, toolContext, args)
	}
	metricRaw, err := execute("get_campaign_metrics")
	if err != nil {
		return nil, err
	}
	benchmarkRaw, err := execute("get_business_benchmark")
	if err != nil {
		return nil, err
	}
	attributionRaw, err := execute("get_attribution_anomalies")
	if err != nil {
		return nil, err
	}
	creativeRaw, err := execute("get_creative_findings")
	if err != nil {
		return nil, err
	}
	researchRaw, err := execute("get_verified_research_sources")
	if err != nil {
		return nil, err
	}
	forecastRaw, err := execute("forecast_ltv")
	if err != nil {
		return nil, err
	}
	var result BusinessInput
	result.TaskID, result.TenantID, result.GameID = taskID, tenantID, gameID
	if json.Unmarshal(metricRaw, &result.Campaign) != nil {
		return nil, fmt.Errorf("decode campaign metric tool output")
	}
	if json.Unmarshal(benchmarkRaw, &result.Benchmarks) != nil {
		return nil, fmt.Errorf("decode benchmark tool output")
	}
	if json.Unmarshal(attributionRaw, &result.AttributionFindings) != nil {
		return nil, fmt.Errorf("decode attribution tool output")
	}
	if json.Unmarshal(creativeRaw, &result.CreativeFindings) != nil {
		return nil, fmt.Errorf("decode creative tool output")
	}
	if json.Unmarshal(researchRaw, &result.ResearchSources) != nil {
		return nil, fmt.Errorf("decode research source tool output")
	}
	if json.Unmarshal(forecastRaw, &result.Forecast) != nil {
		return nil, fmt.Errorf("decode forecast tool output")
	}
	findings, err := s.rules.ListFindings(ctx, tenantID, gameID)
	if err != nil {
		return nil, err
	}
	for _, finding := range findings {
		if finding.CampaignID == campaignID {
			result.RuleFindings = append(result.RuleFindings, finding)
		}
	}
	result.Constraints = []string{"不得直接修改预算", "所有结论必须引用输入数据", "证据不足时必须标记待验证", "不得调用未注册工具", "高风险建议必须人工审批"}
	return &result, nil
}

func (s *Service) getCampaignMetrics(ctx context.Context, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	var args readArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}
	rows, err := s.metrics.Campaigns(ctx, toolContext.TenantID, args.GameID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.CampaignID == args.CampaignID {
			return json.Marshal(row)
		}
	}
	return nil, fmt.Errorf("campaign metrics not found")
}

func (s *Service) getBenchmarks(ctx context.Context, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	var args readArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}
	rows, err := s.rules.ListBenchmarks(ctx, toolContext.TenantID, args.GameID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(rows)
}

func (s *Service) getAttribution(ctx context.Context, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	var args readArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}
	rows, err := s.attr.List(ctx, toolContext.TenantID, args.GameID)
	if err != nil {
		return nil, err
	}
	filtered := rows[:0]
	for _, row := range rows {
		if row.CampaignID == args.CampaignID {
			filtered = append(filtered, row)
		}
	}
	return json.Marshal(filtered)
}

func (s *Service) getCreative(ctx context.Context, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	var args readArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}
	rows, err := s.creative.List(ctx, toolContext.TenantID, args.GameID)
	if err != nil {
		return nil, err
	}
	filtered := rows[:0]
	for _, row := range rows {
		if row.CampaignID == args.CampaignID {
			filtered = append(filtered, row)
		}
	}
	return json.Marshal(filtered)
}

func (s *Service) getVerifiedResearch(ctx context.Context, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	var args readArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}
	if s.research == nil {
		return json.Marshal([]researchdomain.Evidence{})
	}
	through, err := time.Parse("2006-01-02", args.AnalysisDate)
	if err != nil {
		return nil, fmt.Errorf("analysis_date must be YYYY-MM-DD")
	}
	rows, err := s.research.SearchVerified(ctx, toolContext.TenantID, args.GameID, args.CampaignID, through.Add(24*time.Hour-time.Nanosecond), 20)
	if err != nil {
		return nil, err
	}
	result := make([]researchdomain.Evidence, 0, len(rows))
	for _, row := range rows {
		if row.ReviewedAt == nil {
			continue
		}
		result = append(result, researchdomain.Evidence{ID: row.ID, Category: row.Category, Title: row.Title, Summary: row.Summary, SourceURL: row.SourceURL, Publisher: row.Publisher, PublishedAt: row.PublishedAt, VerifiedAt: *row.ReviewedAt})
	}
	return json.Marshal(result)
}

func (s *Service) forecastLTV(ctx context.Context, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	raw, err := s.getCampaignMetrics(ctx, toolContext, input)
	if err != nil {
		return nil, err
	}
	var metric metricsdto.CampaignMetric
	if err := json.Unmarshal(raw, &metric); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"method": "observed_d7_proxy", "ltv_d7": metric.LTVD7, "predicted_revenue_d7": metric.RevenueD7, "requires_more_history": true})
}

func (s *Service) createRecommendation(ctx context.Context, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	var row domain.Recommendation
	if err := json.Unmarshal(input, &row); err != nil {
		return nil, err
	}
	row.ID, row.TenantID, row.TaskID, row.Status = uuid.NewString(), toolContext.TenantID, toolContext.TaskID, "PROPOSED"
	if err := s.repo.CreateRecommendation(ctx, &row); err != nil {
		return nil, err
	}
	return json.Marshal(row)
}

func (s *Service) createApproval(ctx context.Context, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	var args approvalArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}
	recommendation, err := s.repo.GetRecommendation(ctx, toolContext.TenantID, args.RecommendationID)
	if err != nil {
		return nil, err
	}
	row := domain.ApprovalRequest{ID: uuid.NewString(), TenantID: toolContext.TenantID, RecommendationID: args.RecommendationID, TaskID: toolContext.TaskID, ReportID: args.ReportID, Action: recommendation.Action, SuggestedValue: recommendation.SuggestedValue, Reason: recommendation.Description, RiskLevel: recommendation.RiskLevel, Status: "PENDING", RequestedBy: identity.SystemAgentUserID}
	if err := s.repo.CreateApproval(ctx, &row); err != nil {
		return nil, err
	}
	return json.Marshal(row)
}

func (s *Service) persistValidated(ctx context.Context, tenantID, taskID, gameID, campaignID, reportID string, result domain.Result) error {
	for _, finding := range result.Findings {
		evidence, _ := json.Marshal(finding.Evidence)
		row := domain.BusinessFinding{ID: uuid.NewString(), TenantID: tenantID, TaskID: taskID, GameID: gameID, CampaignID: campaignID, Type: finding.Type, RuleCode: finding.RuleCode, Severity: finding.Severity, Conclusion: finding.Conclusion, Description: finding.Description, EvidenceJSON: evidence, Confidence: finding.Confidence}
		if err := s.repo.CreateFinding(ctx, &row); err != nil {
			return err
		}
	}
	toolContext := agenttools.Context{TenantID: tenantID, UserID: identity.SystemAgentUserID, TaskID: taskID}
	for _, output := range result.Recommendations {
		payload, _ := json.Marshal(domain.Recommendation{CampaignID: campaignID, Action: output.Action, Description: output.Description, Priority: output.Priority, RiskLevel: output.RiskLevel, RequiresApproval: output.RequiresApproval, SuggestedValue: output.SuggestedValue})
		created, err := s.executeTool(ctx, "create_recommendation", toolContext, payload)
		if err != nil {
			return err
		}
		if output.RequiresApproval {
			var recommendation domain.Recommendation
			if json.Unmarshal(created, &recommendation) != nil {
				return fmt.Errorf("decode recommendation tool output")
			}
			approvalPayload, _ := json.Marshal(approvalArgs{RecommendationID: recommendation.ID, ReportID: reportID})
			if _, err := s.executeTool(ctx, "create_approval_request", toolContext, approvalPayload); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) executeTool(ctx context.Context, name string, toolContext agenttools.Context, input json.RawMessage) (json.RawMessage, error) {
	output, executionErr := s.registry.Execute(ctx, s.spec.Tools, name, toolContext, input)
	if s.auditor != nil {
		status := "SUCCEEDED"
		errorMessage := ""
		if executionErr != nil {
			status, errorMessage = "FAILED", executionErr.Error()
		}
		if err := s.auditor.Record(ctx, auditdomain.RecordInput{
			TenantID: toolContext.TenantID, ActorID: identity.SystemAgentUserID, ActorType: "SYSTEM_AGENT",
			Action: "AGENT_TOOL_CALL", ResourceType: "AGENT_TOOL", ResourceID: toolContext.TaskID, TaskID: toolContext.TaskID,
			Metadata: map[string]any{"tool": name, "status": status, "error": errorMessage},
		}); err != nil {
			return nil, fmt.Errorf("audit tool call %s: %w", name, err)
		}
	}
	return output, executionErr
}
