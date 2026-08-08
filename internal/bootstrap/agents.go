package bootstrap

import (
	"time"

	agentdomain "github.com/example/adnova/internal/agent/domain"
	agentruntime "github.com/example/adnova/internal/agent/runtime"
	"github.com/example/adnova/internal/agents"
	attributionservice "github.com/example/adnova/internal/attribution/service"
	businessservice "github.com/example/adnova/internal/business/service"
	creativeanalysisservice "github.com/example/adnova/internal/creative/analysis/service"
	dataqualityservice "github.com/example/adnova/internal/dataquality/service"
	metricsservice "github.com/example/adnova/internal/metrics/service"
	researchservice "github.com/example/adnova/internal/research/service"
	rulesservice "github.com/example/adnova/internal/rules/service"
)

func NewAgentRegistry(metrics *metricsservice.Service, rules *rulesservice.Service, attribution *attributionservice.Service, creative *creativeanalysisservice.Service, business *businessservice.Service, dataQuality *dataqualityservice.Service, research *researchservice.Service) (*agentruntime.Registry, error) {
	registry := agentruntime.NewRegistry()
	definitions := []agentruntime.Definition{
		{
			Spec:         agentdomain.AgentSpec{Name: "data-agent", Version: "1.1.0", Description: "数据覆盖与新鲜度校验、确定性指标计算与经营规则物化", ExecutionMode: "SYNCHRONOUS_DETERMINISTIC", Tools: []string{"assess_data_quality", "recalculate_metrics", "materialize_business_rules", "validate_imported_data"}, Permissions: []string{"READ_IMPORTED_DATA", "READ_IMPORT_STATUS", "WRITE_METRICS", "WRITE_ANALYSIS"}, MaxSteps: 5, Timeout: 2 * time.Minute, InputSchema: "full-analysis-input/1.0.0", OutputSchema: "data-agent-output/1.1.0"},
			Availability: agentruntime.AvailabilityReady,
			Executor:     agents.NewDataExecutor(metrics, rules, dataQuality),
		},
		{
			Spec:         agentdomain.AgentSpec{Name: "attribution-agent", Version: "1.1.0", Description: "AppsFlyer、导入 MMP 与渠道归因异常分析", ExecutionMode: "SYNCHRONOUS_DETERMINISTIC", Tools: []string{"calculate_install_gap", "calculate_revenue_gap", "detect_mmp_delay", "persist_attribution_findings"}, Permissions: []string{"READ_CHANNEL_DATA", "READ_MMP_DATA", "READ_GAME_REVENUE", "WRITE_ANALYSIS"}, MaxSteps: 4, Timeout: time.Minute, InputSchema: "full-analysis-input/1.0.0", OutputSchema: "attribution-agent-output/1.1.0"},
			Availability: agentruntime.AvailabilityReady,
			Details:      []string{"file_import_ready", "appsflyer_read_only_sync_ready", "adjust_connector_not_configured"},
			Executor:     agents.NewAttributionExecutor(attribution),
		},
		{
			Spec:         agentdomain.AgentSpec{Name: "creative-agent", Version: "1.1.0", Description: "素材生命周期、CTR衰退、频次与疲劳风险分析", ExecutionMode: "SYNCHRONOUS_DETERMINISTIC", Tools: []string{"calculate_fatigue_score", "detect_ctr_decline", "detect_high_frequency", "detect_high_spend_low_conversion", "persist_creative_findings"}, Permissions: []string{"READ_CREATIVE_METRICS", "WRITE_ANALYSIS"}, MaxSteps: 5, Timeout: time.Minute, InputSchema: "full-analysis-input/1.0.0", OutputSchema: "creative-agent-output/1.1.0"},
			Availability: agentruntime.AvailabilityReady,
			Executor:     agents.NewCreativeExecutor(creative),
		},
		{
			Spec:         agentdomain.AgentSpec{Name: "business-agent", Version: "1.1.0", Description: "ROAS、LTV、付费率和预算风险综合分析", ExecutionMode: "ASYNCHRONOUS_LLM", Tools: business.Capabilities(nil), Permissions: []string{"READ_METRICS", "READ_ANALYSIS", "READ_VERIFIED_RESEARCH", "CREATE_RECOMMENDATION", "CREATE_APPROVAL_REQUEST"}, MaxSteps: 10, Timeout: 2 * time.Minute, InputSchema: "business-agent-input/1.1.0", OutputSchema: "business-agent-output/1.0.0"},
			Availability: agentruntime.AvailabilityReady,
			Executor:     agents.NewBusinessExecutor(business),
		},
		{
			Spec:         agentdomain.AgentSpec{Name: "research-agent", Version: "1.1.0", Description: "读取经人工核验的政策、竞品和市场信息并强制保留来源", ExecutionMode: "VERIFIED_RESEARCH", Tools: []string{"list_verified_sources", "source_contract", "preserve_provenance"}, Permissions: []string{"READ_VERIFIED_PUBLIC_SOURCES"}, MaxSteps: 6, Timeout: 2 * time.Minute, InputSchema: "full-analysis-input/1.0.0", OutputSchema: "research-agent-output/1.1.0"},
			Availability: agentruntime.AvailabilityReady,
			Details:      []string{"manual_source_registration", "human_verification_required", "fabrication_disabled", "live_web_connector_not_configured"},
			Executor:     agents.NewResearchExecutor(research),
		},
		{
			Spec:         agentdomain.AgentSpec{Name: "report-agent", Version: "1.1.0", Description: "整合已验证结论与来源并生成可追溯 Markdown 快照", ExecutionMode: "SYNCHRONOUS_DETERMINISTIC", Tools: []string{"compose_analysis_report", "get_analysis_report", "verify_source_digest"}, Permissions: []string{"READ_AGENT_OUTPUT", "READ_VERIFIED_RESEARCH", "READ_REPORT"}, MaxSteps: 3, Timeout: time.Minute, InputSchema: "report-agent-input/1.0.0", OutputSchema: "report-agent-output/1.1.0"},
			Availability: agentruntime.AvailabilityReady,
			Executor:     agents.NewReportExecutor(business),
		},
		{
			Spec:         agentdomain.AgentSpec{Name: "openclaw-agent", Version: "1.2.0", Description: "用户交互、工作流启动、审批入口与消息适配边界", ExecutionMode: "INTERACTION_GATEWAY", Tools: []string{"start_analysis_workflow", "get_workflow_status", "list_pending_approvals", "list_notifications", "mark_notification_read"}, Permissions: []string{"START_WORKFLOW", "READ_WORKFLOW", "READ_APPROVAL", "READ_NOTIFICATION", "MARK_NOTIFICATION_READ"}, MaxSteps: 5, Timeout: 30 * time.Second, InputSchema: "openclaw-command/1.1.0", OutputSchema: "openclaw-agent-output/1.1.0"},
			Availability: agentruntime.AvailabilityReady,
			Details:      []string{"internal_api_ready", "internal_notification_ready", "external_message_push_not_configured", "no_ad_platform_execution"},
			Executor:     agents.NewOpenClawExecutor(),
		},
	}
	for _, definition := range definitions {
		if err := registry.Register(definition); err != nil {
			return nil, err
		}
	}
	return registry, nil
}
