package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	businessdomain "github.com/example/adnova/internal/business/domain"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/google/uuid"
)

const (
	GeneratorAgent   = "report-agent"
	GeneratorVersion = "1.2.0"
)

type Input struct {
	TenantID     string
	TaskID       string
	WorkflowID   string
	GameID       string
	CampaignID   string
	Result       businessdomain.Result
	Research     []researchdomain.Evidence
	AnalysisDate string
	TraceID      string
}

func Compose(input Input) *businessdomain.AnalysisReport {
	return ComposeWithPolisher(context.Background(), input, nil)
}

func ComposeWithPolisher(ctx context.Context, input Input, polisher Polisher) *businessdomain.AnalysisReport {
	sourcePayload := sourcePayload(input)
	digest := sha256.Sum256(sourcePayload)
	digestText := hex.EncodeToString(digest[:])
	summary := input.Result.Summary
	enhancement := map[string]any{"status": "DISABLED", "fact_source": "DETERMINISTIC"}
	if polisher != nil && polisher.Enabled() {
		polished, err := polisher.Polish(ctx, input, digestText)
		if err == nil {
			summary = polished.Summary
			enhancement = map[string]any{
				"status": "APPLIED", "fact_source": "DETERMINISTIC", "provider": polished.Provider,
				"model": polished.Model, "prompt_version": polished.PromptVersion, "schema_version": polished.SchemaVersion,
			}
		} else {
			enhancement = map[string]any{"status": "FALLBACK", "fact_source": "DETERMINISTIC", "error_category": FailureCategory(err)}
		}
	}
	generatedAt := time.Now().UTC()
	provenance, _ := json.Marshal(map[string]any{
		"workflow_id": input.WorkflowID, "task_id": input.TaskID,
		"generator_agent": GeneratorAgent, "generator_version": GeneratorVersion,
		"research_source_ids": sourceIDs(input.Research), "source_digest_algorithm": "SHA-256",
		"summary_enhancement": enhancement,
	})

	var body strings.Builder
	fmt.Fprintf(&body, "# 经营分析报告\n\n%s\n\n## 经营发现\n\n", summary)
	if len(input.Result.Findings) == 0 {
		body.WriteString("无经营风险发现。\n")
	}
	for _, finding := range input.Result.Findings {
		fmt.Fprintf(&body, "- **%s [%s]**：%s\n", escapeText(finding.Conclusion), escapeText(finding.Severity), escapeText(finding.Description))
	}
	body.WriteString("\n## 建议\n\n")
	if len(input.Result.Recommendations) == 0 {
		body.WriteString("无新增建议。\n")
	}
	for _, recommendation := range input.Result.Recommendations {
		fmt.Fprintf(&body, "- **%s**：%s（风险：%s，人工审批：%t）\n", escapeText(recommendation.Action), escapeText(recommendation.Description), escapeText(recommendation.RiskLevel), recommendation.RequiresApproval)
	}
	body.WriteString("\n## 已核验研究来源\n\n")
	if len(input.Research) == 0 {
		body.WriteString("本次分析没有可用的已核验外部研究来源。\n")
	}
	for _, source := range input.Research {
		fmt.Fprintf(&body, "- [%s](%s) — %s，%s\n", escapeText(source.Title), source.SourceURL, escapeText(source.Publisher), source.PublishedAt.Format("2006-01-02"))
	}
	body.WriteString("\n> 本报告只包含建议，不代表广告平台动作已执行。外部研究内容仅来自已核验来源。\n")

	return &businessdomain.AnalysisReport{
		ID: uuid.NewString(), TenantID: input.TenantID, TaskID: input.TaskID,
		Title: "经营分析报告", Summary: summary, ContentMarkdown: body.String(), Status: "READY",
		GeneratorAgent: GeneratorAgent, GeneratorVersion: GeneratorVersion,
		SourceDigest: digestText, ProvenanceJSON: provenance, GeneratedAt: &generatedAt,
	}
}

func sourcePayload(input Input) json.RawMessage {
	payload, _ := json.Marshal(map[string]any{
		"task_id": input.TaskID, "workflow_id": input.WorkflowID, "game_id": input.GameID,
		"campaign_id": input.CampaignID, "analysis_date": input.AnalysisDate,
		"result": input.Result, "research": input.Research,
	})
	return payload
}

func sourceIDs(rows []researchdomain.Evidence) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result
}

func escapeText(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "[", "\\[")
	value = strings.ReplaceAll(value, "]", "\\]")
	return value
}
