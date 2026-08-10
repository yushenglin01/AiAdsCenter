package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	"github.com/example/adnova/internal/agent/intelligence"
	businessservice "github.com/example/adnova/internal/business/service"
	creativedomain "github.com/example/adnova/internal/creative/domain"
	dataqualitydomain "github.com/example/adnova/internal/dataquality/domain"
	researchdomain "github.com/example/adnova/internal/research/domain"
	"github.com/example/adnova/internal/research/websearch"
)

type MetricCalculator interface {
	Recalculate(ctx context.Context, tenantID, gameID string) (int, error)
}

type Analyzer interface {
	Analyze(ctx context.Context, tenantID, gameID string) (int, error)
}

type CreativeFindingReader interface {
	List(ctx context.Context, tenantID, gameID string) ([]creativedomain.Finding, error)
}

type DataProfiler interface {
	Assess(ctx context.Context, tenantID, gameID, analysisDate string) (dataqualitydomain.Report, error)
}

type ResearchReader interface {
	SearchVerified(ctx context.Context, tenantID, gameID, campaignID, analysisDate string) ([]researchdomain.Evidence, error)
}

type ResearchWebCapability interface {
	WebCapability() websearch.Capability
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

type CreativeExecutor struct {
	analyzer Analyzer
	reader   CreativeFindingReader
	runtime  *intelligence.Runtime
	contract intelligence.Contract
}

type CreativeInsight struct {
	FindingID      string   `json:"finding_id"`
	Interpretation string   `json:"interpretation"`
	PossibleCauses []string `json:"possible_causes"`
	NextChecks     []string `json:"next_checks"`
}

func NewCreativeExecutor(analyzer Analyzer) *CreativeExecutor {
	reader, _ := analyzer.(CreativeFindingReader)
	return &CreativeExecutor{analyzer: analyzer, reader: reader}
}

func NewCreativeExecutorWithIntelligence(analyzer Analyzer, runtime *intelligence.Runtime, contract intelligence.Contract) *CreativeExecutor {
	result := NewCreativeExecutor(analyzer)
	result.runtime, result.contract = runtime, contract
	return result
}

func (e *CreativeExecutor) Execute(ctx context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	payload, err := decodePayload(input.Payload)
	if err != nil {
		return nil, err
	}
	count, err := e.analyzer.Analyze(ctx, input.TenantID, payload.GameID)
	if err != nil {
		return nil, err
	}
	base := map[string]any{
		"findings": count, "calculation_mode": "DETERMINISTIC",
		"analysis_capabilities": []string{"fatigue_score", "lifecycle_signals", "multimodal_asset_input_not_configured"},
	}
	if e.reader == nil || e.runtime == nil || !e.runtime.Enabled("creative-agent") || count == 0 {
		output, _ := json.Marshal(base)
		return result(input, "creative-agent", "SUCCEEDED", output), nil
	}
	rows, err := e.reader.List(ctx, input.TenantID, payload.GameID)
	if err != nil {
		return nil, err
	}
	findings := make([]creativedomain.Finding, 0, len(rows))
	for _, row := range rows {
		if payload.CampaignID == "" || row.CampaignID == payload.CampaignID {
			findings = append(findings, row)
		}
	}
	if len(findings) == 0 {
		output, _ := json.Marshal(base)
		return result(input, "creative-agent", "SUCCEEDED", output), nil
	}
	structuredInput, _ := json.Marshal(map[string]any{"findings": findings, "media_assets": []any{}})
	generated, generationErr := e.runtime.Generate(ctx, intelligence.Request{
		AgentName: "creative-agent", TenantID: input.TenantID, TaskID: input.TaskID, TraceID: input.TraceID,
		Contract: e.contract, UserPrompt: "解释确定性素材发现，给出待验证原因和只读核查项。",
		InputJSON: structuredInput, Temperature: 0.1, MaxTokens: 1600,
	})
	if generationErr != nil {
		base["calculation_mode"] = "DETERMINISTIC_FALLBACK"
		base["llm"] = map[string]any{"status": "FALLBACK", "error_category": intelligence.FailureCategory(generationErr)}
		output, _ := json.Marshal(base)
		return result(input, "creative-agent", "SUCCEEDED", output), nil
	}
	var llmOutput struct {
		Insights []CreativeInsight `json:"insights"`
	}
	validationErr := json.Unmarshal(generated.Content, &llmOutput)
	if validationErr == nil {
		validationErr = validateCreativeInsights(llmOutput.Insights, findings)
	}
	if validationErr != nil {
		e.runtime.MarkValidationFailed(ctx, generated)
		base["calculation_mode"] = "DETERMINISTIC_FALLBACK"
		base["llm"] = map[string]any{"status": "FALLBACK", "error_category": "VALIDATION_FAILED"}
	} else {
		base["calculation_mode"] = "HYBRID_DETERMINISTIC_LLM"
		base["insights"] = llmOutput.Insights
		base["llm"] = map[string]any{
			"status": "APPLIED", "provider": generated.Provider, "model": generated.Model,
			"prompt_version": generated.PromptVersion, "schema_version": generated.SchemaVersion,
		}
	}
	output, _ := json.Marshal(base)
	return result(input, "creative-agent", "SUCCEEDED", output), nil
}

func (e *CreativeExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	provider := "deterministic"
	details := []string{"fatigue_score", "lifecycle_signals", "multimodal_asset_input_not_configured"}
	if e.runtime != nil && e.runtime.Enabled("creative-agent") {
		provider += "+" + e.runtime.Provider()
		details = append(details, "llm_explanation_ready")
	}
	return &agentdomain.HealthStatus{Status: "UP", Provider: provider, Details: details}, nil
}

func (e *CreativeExecutor) Capabilities(context.Context) []string {
	result := []string{"fatigue_score", "lifecycle_signals", "multimodal_asset_input_not_configured"}
	if e.runtime != nil && e.runtime.Enabled("creative-agent") {
		result = append(result, "explain_deterministic_findings")
	}
	return result
}

func validateCreativeInsights(insights []CreativeInsight, findings []creativedomain.Finding) error {
	allowed := make(map[string]bool, len(findings))
	for _, finding := range findings {
		allowed[finding.ID] = true
	}
	seen := map[string]bool{}
	if len(insights) > 30 {
		return fmt.Errorf("too many creative insights")
	}
	for _, insight := range insights {
		if !allowed[insight.FindingID] || seen[insight.FindingID] || strings.TrimSpace(insight.Interpretation) == "" {
			return fmt.Errorf("creative insight is not grounded in a deterministic finding")
		}
		seen[insight.FindingID] = true
	}
	return nil
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

type ReportExecutor struct {
	service *businessservice.Service
	runtime *intelligence.Runtime
}

func NewReportExecutor(service *businessservice.Service) *ReportExecutor {
	return &ReportExecutor{service: service}
}

func NewReportExecutorWithIntelligence(service *businessservice.Service, runtime *intelligence.Runtime) *ReportExecutor {
	return &ReportExecutor{service: service, runtime: runtime}
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
	provider := "deterministic"
	details := []string{"immutable_markdown", "source_preserving", "deterministic_fact_sections"}
	if e.runtime != nil && e.runtime.Enabled("report-agent") {
		provider += "+" + e.runtime.Provider()
		details = append(details, "optional_summary_polish_ready")
	}
	return &agentdomain.HealthStatus{Status: "UP", Provider: provider, Details: details}, nil
}
func (e *ReportExecutor) Capabilities(context.Context) []string {
	return []string{"get_analysis_report", "format_markdown_snapshot"}
}

type ResearchExecutor struct {
	reader   ResearchReader
	web      ResearchWebCapability
	runtime  *intelligence.Runtime
	contract intelligence.Contract
}

type ResearchSignal struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	SourceIDs   []string `json:"source_ids"`
	Confidence  float64  `json:"confidence"`
}

type ResearchSynthesis struct {
	Summary string           `json:"summary"`
	Signals []ResearchSignal `json:"signals"`
}

func NewResearchExecutor(readers ...ResearchReader) *ResearchExecutor {
	result := &ResearchExecutor{}
	if len(readers) > 0 {
		result.reader = readers[0]
		result.web, _ = readers[0].(ResearchWebCapability)
	}
	return result
}

func NewResearchExecutorWithIntelligence(reader ResearchReader, runtime *intelligence.Runtime, contract intelligence.Contract) *ResearchExecutor {
	result := NewResearchExecutor(reader)
	result.runtime, result.contract = runtime, contract
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
	base := map[string]any{
		"status": status, "items": items, "source_policy": "VERIFIED_ONLY",
		"fabrication_disabled": true, "reasoning_mode": "DETERMINISTIC_SOURCE_SELECTION",
	}
	if len(items) > 0 && e.runtime != nil && e.runtime.Enabled("research-agent") {
		structuredInput, _ := json.Marshal(map[string]any{
			"game_id": payload.GameID, "campaign_id": payload.CampaignID,
			"analysis_date": normalizedAnalysisDate(payload.AnalysisDate), "verified_sources": items,
		})
		generated, generationErr := e.runtime.Generate(ctx, intelligence.Request{
			AgentName: "research-agent", TenantID: input.TenantID, TaskID: input.TaskID, TraceID: input.TraceID,
			Contract: e.contract, UserPrompt: "归纳已核验来源中的关键信号，并为每条信号列出 source_id。",
			InputJSON: structuredInput, Temperature: 0.1, MaxTokens: 1600,
		})
		if generationErr != nil {
			base["reasoning_mode"] = "DETERMINISTIC_FALLBACK"
			base["llm"] = map[string]any{"status": "FALLBACK", "error_category": intelligence.FailureCategory(generationErr)}
		} else {
			var synthesis ResearchSynthesis
			validationErr := json.Unmarshal(generated.Content, &synthesis)
			if validationErr == nil {
				validationErr = validateResearchSynthesis(synthesis, items)
			}
			if validationErr != nil {
				e.runtime.MarkValidationFailed(ctx, generated)
				base["reasoning_mode"] = "DETERMINISTIC_FALLBACK"
				base["llm"] = map[string]any{"status": "FALLBACK", "error_category": "VALIDATION_FAILED"}
			} else {
				base["reasoning_mode"] = "VERIFIED_SOURCE_LLM_SYNTHESIS"
				base["synthesis"] = synthesis
				base["llm"] = map[string]any{
					"status": "APPLIED", "provider": generated.Provider, "model": generated.Model,
					"prompt_version": generated.PromptVersion, "schema_version": generated.SchemaVersion,
				}
			}
		}
	}
	output, _ := json.Marshal(base)
	return result(input, "research-agent", "SUCCEEDED", output), nil
}
func (e *ResearchExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	if e.reader == nil {
		return &agentdomain.HealthStatus{Status: "DEGRADED", Provider: "none", Details: []string{"verified_source_repository_not_configured"}}, nil
	}
	details := []string{"manual_source_registration", "human_verification", "source_preserving"}
	provider := "verified-source-repository"
	if e.web != nil {
		capability := e.web.WebCapability()
		if capability.Configured {
			provider += "+" + capability.Provider
			details = append(details, "live_web_search_ready")
		} else {
			details = append(details, "live_web_connector_not_configured")
		}
	}
	if e.runtime != nil && e.runtime.Enabled("research-agent") {
		provider += "+" + e.runtime.Provider()
		details = append(details, "verified_source_llm_synthesis_ready")
	}
	return &agentdomain.HealthStatus{Status: "UP", Provider: provider, Details: details}, nil
}
func (e *ResearchExecutor) Capabilities(context.Context) []string {
	capabilities := []string{"list_verified_sources", "source_contract", "preserve_provenance"}
	if e.web != nil && e.web.WebCapability().Configured {
		capabilities = append(capabilities, "search_public_web")
	}
	if e.web != nil && e.web.WebCapability().ImportEnabled {
		capabilities = append(capabilities, "import_web_result_for_review")
	}
	if e.runtime != nil && e.runtime.Enabled("research-agent") {
		capabilities = append(capabilities, "synthesize_verified_sources")
	}
	return capabilities
}

func validateResearchSynthesis(synthesis ResearchSynthesis, items []researchdomain.Evidence) error {
	if strings.TrimSpace(synthesis.Summary) == "" || len(synthesis.Signals) > 12 {
		return fmt.Errorf("research synthesis summary or signal count is invalid")
	}
	allowed := make(map[string]bool, len(items))
	for _, item := range items {
		allowed[item.ID] = true
	}
	for _, signal := range synthesis.Signals {
		if strings.TrimSpace(signal.Title) == "" || strings.TrimSpace(signal.Description) == "" || len(signal.SourceIDs) == 0 || signal.Confidence < 0 || signal.Confidence > 1 {
			return fmt.Errorf("research signal is incomplete")
		}
		for _, sourceID := range signal.SourceIDs {
			if !allowed[sourceID] {
				return fmt.Errorf("research signal references an unverified source")
			}
		}
	}
	return nil
}

type OpenClawExecutor struct{ runtime *intelligence.Runtime }

func NewOpenClawExecutor() *OpenClawExecutor { return &OpenClawExecutor{} }
func NewOpenClawExecutorWithIntelligence(runtime *intelligence.Runtime) *OpenClawExecutor {
	return &OpenClawExecutor{runtime: runtime}
}
func (e *OpenClawExecutor) Execute(_ context.Context, input agentdomain.AgentInput) (*agentdomain.AgentResult, error) {
	output, _ := json.Marshal(map[string]any{"channel": "internal_api", "workflow_start": "READY", "workflow_status": "READY", "approval_inbox": "READY", "notification_inbox": "READY", "external_delivery": "NOT_CONFIGURED", "ad_platform_execution": "FORBIDDEN"})
	return result(input, "openclaw-agent", "SUCCEEDED", output), nil
}
func (e *OpenClawExecutor) Health(context.Context) (*agentdomain.HealthStatus, error) {
	provider := "internal-api"
	details := []string{"workflow_commands", "approval_inbox", "notification_inbox"}
	if e.runtime != nil && e.runtime.Enabled("openclaw-agent") {
		provider += "+" + e.runtime.Provider()
		details = append(details, "natural_language_intent_parser_ready", "write_intent_confirmation_required")
	}
	return &agentdomain.HealthStatus{Status: "UP", Provider: provider, Details: details}, nil
}
func (e *OpenClawExecutor) Capabilities(context.Context) []string {
	result := []string{"start_analysis_workflow", "get_workflow_status", "list_pending_approvals", "list_notifications", "mark_notification_read"}
	if e.runtime != nil && e.runtime.Enabled("openclaw-agent") {
		result = append(result, "parse_natural_language_command")
	}
	return result
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
