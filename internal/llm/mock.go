package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type MockClient struct{ Model string }

func NewMock(model string) *MockClient {
	if model == "" {
		model = "mock-business-v1"
	}
	return &MockClient{Model: model}
}

func (m *MockClient) Name() string { return "mock" }

func (m *MockClient) GenerateStructured(_ context.Context, req GenerateRequest) (*GenerateResponse, error) {
	started := time.Now()
	var input struct {
		Campaign struct {
			ID   string `json:"campaign_id"`
			Name string `json:"campaign_name"`
		} `json:"campaign"`
		RuleFindings []struct {
			RuleCode    string         `json:"rule_code"`
			Severity    string         `json:"severity"`
			Title       string         `json:"title"`
			Description string         `json:"description"`
			Evidence    map[string]any `json:"evidence"`
		} `json:"rule_findings"`
		AttributionFindings []struct {
			RuleCode    string         `json:"rule_code"`
			Severity    string         `json:"severity"`
			Title       string         `json:"title"`
			Description string         `json:"description"`
			Evidence    map[string]any `json:"evidence"`
		} `json:"attribution_findings"`
		CreativeFindings []struct {
			RuleCode    string         `json:"rule_code"`
			Severity    string         `json:"severity"`
			Title       string         `json:"title"`
			Description string         `json:"description"`
			Evidence    map[string]any `json:"evidence"`
		} `json:"creative_findings"`
	}
	if err := json.Unmarshal(req.InputJSON, &input); err != nil {
		return nil, &ClientError{Category: ErrorInvalidRequest, Message: "invalid structured input", Retryable: false}
	}
	findings := make([]map[string]any, 0)
	appendFinding := func(kind, code, severity, conclusion, description string, evidence map[string]any) {
		findings = append(findings, map[string]any{"type": kind, "rule_code": code, "severity": severity, "conclusion": conclusion, "description": description, "evidence": evidenceList(evidence), "possible_causes": []string{"投放流量结构或素材表现发生变化"}, "impact": "可能影响预算回收效率", "confidence": 0.92})
	}
	for _, finding := range input.RuleFindings {
		appendFinding("business_risk", finding.RuleCode, finding.Severity, finding.Title, finding.Description, finding.Evidence)
	}
	for _, finding := range input.AttributionFindings {
		appendFinding("attribution_risk", finding.RuleCode, finding.Severity, finding.Title, finding.Description, finding.Evidence)
	}
	for _, finding := range input.CreativeFindings {
		if finding.RuleCode == "CREATIVE_FATIGUE_HIGH" {
			appendFinding("creative_fatigue", finding.RuleCode, finding.Severity, finding.Title, finding.Description, finding.Evidence)
		}
	}
	recommendations := []map[string]any{}
	if len(input.RuleFindings) > 0 {
		recommendations = append(recommendations, map[string]any{"action": "REDUCE_BUDGET", "description": "建议暂时降低该计划 20% 预算，观察 24 小时后复核", "priority": 1, "risk_level": "HIGH", "requires_approval": true, "suggested_value": "0.20"})
	}
	if len(input.CreativeFindings) > 0 {
		recommendations = append(recommendations, map[string]any{"action": "REPLACE_CREATIVE", "description": "建议准备替换疲劳素材，不自动上传或启用素材", "priority": 2, "risk_level": "HIGH", "requires_approval": true})
	}
	if len(input.AttributionFindings) > 0 {
		recommendations = append(recommendations, map[string]any{"action": "VERIFY_ATTRIBUTION", "description": "建议核查渠道与 MMP 的归因窗口和回传延迟", "priority": 3, "risk_level": "MEDIUM", "requires_approval": false})
	}
	recommendations = append(recommendations, map[string]any{"action": "OBSERVE", "description": "保持正常对照计划并观察 24 小时", "priority": 4, "risk_level": "LOW", "requires_approval": false})
	status := "HEALTHY"
	if len(findings) > 0 {
		status = "CRITICAL"
	}
	output := map[string]any{"status": status, "summary": fmt.Sprintf("%s 共识别 %d 个需关注信号，所有建议均未直接执行。", input.Campaign.Name, len(findings)), "findings": findings, "recommendations": recommendations}
	content, _ := json.Marshal(output)
	return &GenerateResponse{Content: content, Model: m.Model, FinishReason: "stop", Usage: Usage{InputTokens: estimateTokens(req.InputJSON), OutputTokens: estimateTokens(content)}, Latency: time.Since(started)}, nil
}

func evidenceList(values map[string]any) []map[string]any {
	if metric, ok := values["metric"].(string); ok {
		if actual, exists := values["actual"]; exists {
			item := map[string]any{"metric": metric, "actual": fmt.Sprint(actual)}
			if target, exists := values["threshold"]; exists {
				item["target"] = fmt.Sprint(target)
			}
			return []map[string]any{item}
		}
	}
	result := make([]map[string]any, 0, len(values))
	keys := make([]string, 0, len(values))
	for metric := range values {
		keys = append(keys, metric)
	}
	sort.Strings(keys)
	for _, metric := range keys {
		value := values[metric]
		if metric == "threshold" || metric == "target_roas" {
			continue
		}
		item := map[string]any{"metric": metric, "actual": fmt.Sprint(value)}
		if target, ok := values["threshold"]; ok && metric == primaryThresholdMetric(values) {
			item["target"] = fmt.Sprint(target)
		} else if target, ok := values["target_roas"]; ok && metric == "roas_d7" {
			item["target"] = fmt.Sprint(target)
		}
		result = append(result, item)
	}
	if len(result) == 0 {
		result = append(result, map[string]any{"metric": "requires_verification", "actual": "true"})
	}
	return result
}

func primaryThresholdMetric(values map[string]any) string {
	for _, metric := range []string{"fatigue_score", "cpi", "payer_rate", "difference_rate", "frequency", "ctr_decline"} {
		if _, exists := values[metric]; exists {
			return metric
		}
	}
	return ""
}

func estimateTokens(value []byte) int {
	if len(value) == 0 {
		return 0
	}
	return (len(value) + 3) / 4
}
