package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	businessservice "github.com/example/adnova/internal/business/service"
	dataqualitydomain "github.com/example/adnova/internal/dataquality/domain"
	researchdomain "github.com/example/adnova/internal/research/domain"
)

type MetricCalculator interface {
	Recalculate(ctx context.Context, tenantID, gameID string) (int, error)
}

type Analyzer interface {
	Analyze(ctx context.Context, tenantID, gameID string) (int, error)
}

type DataProfiler interface {
	Assess(ctx context.Context, tenantID, gameID, analysisDate string) (dataqualitydomain.Report, error)
}

type ResearchReader interface {
	SearchVerified(ctx context.Context, tenantID, gameID, campaignID, analysisDate string) ([]researchdomain.Evidence, error)
}

type AnalysisPayload struct {
	GameID       string `json:"game_id"`
	CampaignID   string `json:"campaign_id"`
	AnalysisDate string `json:"analysis_date,omitempty"`
	BusinessTask string `json:"business_task_id,omitempty"`
}

type DataExecutor struct {
	metrics MetricCalculator
	rules   Analyzer
	profile DataProfiler
}

func NewDataExecutor(metrics MetricCalculator, rules Analyzer, profilers ...DataProfiler) *DataExecutor {
	result := &DataExecutor{metrics: metrics, rules: rules}
	if len(profilers) > 0 {
		result.profile = profilers[0]
	}
	return result
}

func (e *DataExecutor) Execute(ctx context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	payload, err := decodePayload(input.Payload)
	if err != nil {
		return nil, err
	}
	analysisDate := normalizedAnalysisDate(payload.AnalysisDate)
	quality := dataqualitydomain.Report{GameID: payload.GameID, AnalysisDate: analysisDate, Status: "NOT_CONFIGURED", Warnings: []string{"data quality profiler is not configured"}}
	if e.profile != nil {
		quality, err = e.profile.Assess(ctx, input.TenantID, payload.GameID, analysisDate)
		if err != nil {
			return nil, err
		}
	}
	count, err := e.metrics.Recalculate(ctx, input.TenantID, payload.GameID)
	if err != nil {
		return nil, err
	}
	ruleFindings, err := e.rules.Analyze(ctx, input.TenantID, payload.GameID)
	if err != nil {
		return nil, err
	}
	output, _ := json.Marshal(map[string]any{"calculated_rows": count, "business_rule_findings": ruleFindings, "data_quality": quality, "acquisition_mode": "PREVIOUSLY_IMPORTED_OR_SYNCED", "external_connectors": "NOT_INVOKED"})
	return result(input, "data-agent", "SUCCEEDED", output), nil
}

func (e *DataExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	return &agentdomain.HealthStatus{Status: "UP", Provider: "deterministic", Details: []string{"metric_recalculation", "decimal_safe_math"}}, nil
}
func (e *DataExecutor) Capabilities(context.Context) []string {
	return []string{"recalculate_metrics", "materialize_business_rules", "validate_imported_data"}
}

type AnalysisExecutor struct {
	name     string
	analyzer Analyzer
	details  []string
}

func NewAttributionExecutor(analyzer Analyzer) *AnalysisExecutor {
	return &AnalysisExecutor{name: "attribution-agent", analyzer: analyzer, details: []string{"attribution_gap", "evidence_persistence"}}
}

func NewCreativeExecutor(analyzer Analyzer) *AnalysisExecutor {
	return &AnalysisExecutor{name: "creative-agent", analyzer: analyzer, details: []string{"fatigue_score", "lifecycle_signals"}}
}

func (e *AnalysisExecutor) Execute(ctx context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	payload, err := decodePayload(input.Payload)
	if err != nil {
		return nil, err
	}
	count, err := e.analyzer.Analyze(ctx, input.TenantID, payload.GameID)
	if err != nil {
		return nil, err
	}
	output, _ := json.Marshal(map[string]any{"findings": count, "calculation_mode": "DETERMINISTIC", "analysis_capabilities": e.details})
	return result(input, e.name, "SUCCEEDED", output), nil
}

func (e *AnalysisExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	return &agentdomain.HealthStatus{Status: "UP", Provider: "deterministic", Details: append([]string(nil), e.details...)}, nil
}
func (e *AnalysisExecutor) Capabilities(context.Context) []string {
	return append([]string(nil), e.details...)
}

type BusinessExecutor struct{ service *businessservice.Service }

func NewBusinessExecutor(service *businessservice.Service) *BusinessExecutor {
	return &BusinessExecutor{service: service}
}

func (e *BusinessExecutor) Execute(ctx context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	payload, err := decodePayload(input.Payload)
	if err != nil {
		return nil, err
	}
	details, err := e.service.SubmitForWorkflow(ctx, input.TenantID, input.UserID, input.TraceID, input.WorkflowID, businessservice.AnalyzeInput{GameID: payload.GameID, CampaignID: payload.CampaignID, AnalysisDate: payload.AnalysisDate})
	if err != nil {
		return nil, err
	}
	output, _ := json.Marshal(map[string]any{"business_task_id": details.Task.ID, "status": details.Task.Status})
	row := result(input, "business-agent", "QUEUED", output)
	row.ExternalTaskID = details.Task.ID
	return row, nil
}

func (e *BusinessExecutor) Health(ctx context.Context) (*agentdomain.HealthStatus, error) {
	return e.service.Health(ctx)
}
func (e *BusinessExecutor) Capabilities(ctx context.Context) []string {
	return e.service.Capabilities(ctx)
}

type ReportExecutor struct{ service *businessservice.Service }

func NewReportExecutor(service *businessservice.Service) *ReportExecutor {
	return &ReportExecutor{service: service}
}

func (e *ReportExecutor) Execute(ctx context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	payload, err := decodePayload(input.Payload)
	if err != nil {
		return nil, err
	}
	if payload.BusinessTask == "" {
		return nil, fmt.Errorf("business_task_id is required")
	}
	report, err := e.service.GetReport(ctx, input.TenantID, payload.BusinessTask)
	if err != nil {
		return nil, err
	}
	output, _ := json.Marshal(map[string]any{"report_id": report.ID, "title": report.Title, "summary": report.Summary, "status": report.Status, "generator_agent": report.GeneratorAgent, "generator_version": report.GeneratorVersion, "source_digest": report.SourceDigest, "provenance": report.ProvenanceJSON})
	return result(input, "report-agent", "SUCCEEDED", output), nil
}

func (e *ReportExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	return &agentdomain.HealthStatus{Status: "UP", Provider: "deterministic", Details: []string{"immutable_markdown", "source_preserving"}}, nil
}
func (e *ReportExecutor) Capabilities(context.Context) []string {
	return []string{"get_analysis_report", "format_markdown_snapshot"}
}

type ResearchExecutor struct{ reader ResearchReader }

func NewResearchExecutor(readers ...ResearchReader) *ResearchExecutor {
	result := &ResearchExecutor{}
	if len(readers) > 0 {
		result.reader = readers[0]
	}
	return result
}

func (e *ResearchExecutor) Execute(ctx context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	if e.reader == nil {
		output, _ := json.Marshal(map[string]any{"status": "NOT_CONFIGURED", "items": []any{}, "reason": "No approved research source repository is configured; no market facts were fabricated."})
		return result(input, "research-agent", "SKIPPED", output), nil
	}
	payload, err := decodePayload(input.Payload)
	if err != nil {
		return nil, err
	}
	items, err := e.reader.SearchVerified(ctx, input.TenantID, payload.GameID, payload.CampaignID, normalizedAnalysisDate(payload.AnalysisDate))
	if err != nil {
		return nil, err
	}
	status := "VERIFIED_SOURCES_READY"
	if len(items) == 0 {
		status = "NO_VERIFIED_SOURCES"
	}
	output, _ := json.Marshal(map[string]any{"status": status, "items": items, "source_policy": "VERIFIED_ONLY", "fabrication_disabled": true})
	return result(input, "research-agent", "SUCCEEDED", output), nil
}
func (e *ResearchExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	if e.reader == nil {
		return &agentdomain.HealthStatus{Status: "DEGRADED", Provider: "none", Details: []string{"verified_source_repository_not_configured"}}, nil
	}
	return &agentdomain.HealthStatus{Status: "UP", Provider: "verified-source-repository", Details: []string{"manual_source_registration", "human_verification", "source_preserving"}}, nil
}
func (e *ResearchExecutor) Capabilities(context.Context) []string {
	return []string{"list_verified_sources", "source_contract", "preserve_provenance"}
}

type OpenClawExecutor struct{}

func NewOpenClawExecutor() *OpenClawExecutor { return &OpenClawExecutor{} }
func (e *OpenClawExecutor) Execute(_ context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	output, _ := json.Marshal(map[string]any{"channel": "internal_api", "workflow_start": "READY", "workflow_status": "READY", "approval_inbox": "READY", "notification_inbox": "READY", "external_delivery": "NOT_CONFIGURED", "ad_platform_execution": "FORBIDDEN"})
	return result(input, "openclaw-agent", "SUCCEEDED", output), nil
}
func (e *OpenClawExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	return &agentdomain.HealthStatus{Status: "UP", Provider: "internal-api", Details: []string{"workflow_commands", "approval_inbox", "notification_inbox"}}, nil
}
func (e *OpenClawExecutor) Capabilities(context.Context) []string {
	return []string{"start_analysis_workflow", "get_workflow_status", "list_pending_approvals", "list_notifications", "mark_notification_read"}
}

func decodePayload(data json.RawMessage) (AnalysisPayload, error) {
	var payload AnalysisPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, fmt.Errorf("decode agent input: %w", err)
	}
	if payload.GameID == "" {
		return payload, fmt.Errorf("game_id is required")
	}
	return payload, nil
}

func normalizedAnalysisDate(value string) string {
	if value == "" {
		return time.Now().UTC().Format("2006-01-02")
	}
	return value
}

func result(input agentdomain.AgentInput, name, status string, output json.RawMessage) *agentdomain.AgentResult {
	return &agentdomain.AgentResult{TaskID: input.TaskID, AgentName: name, Status: status, Output: output}
}
