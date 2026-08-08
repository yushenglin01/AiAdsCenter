export interface AgentTask {
  task_id: string
  game_id: string
  campaign_id: string
  task_type: string
  status: string
  current_step: string
  attempt: number
  max_attempts: number
  queue_retry_count: number
  queued_at?: string
  last_heartbeat_at?: string
  output_json?: BusinessResult
  error_message?: string
  created_at: string
  finished_at?: string
}

export interface BusinessEvidence { metric: string; actual: string; target?: string }
export interface BusinessFindingOutput { type: string; rule_code: string; severity: string; conclusion: string; description: string; evidence: BusinessEvidence[]; possible_causes: string[]; impact: string; confidence: number }
export interface BusinessRecommendationOutput { action: string; description: string; priority: number; risk_level: string; requires_approval: boolean; suggested_value?: string }
export interface BusinessResult { status: string; summary: string; findings: BusinessFindingOutput[]; recommendations: BusinessRecommendationOutput[] }

export interface BusinessFinding { id: string; rule_code: string; severity: string; conclusion: string; description: string; evidence: BusinessEvidence[]; confidence: number }
export interface Recommendation { id: string; task_id: string; campaign_id: string; action: string; description: string; priority: number; risk_level: string; requires_approval: boolean; suggested_value?: string; status: string; created_at: string }
export interface ApprovalRequest { id: string; recommendation_id: string; task_id: string; report_id?: string; action: string; suggested_value?: string; reason?: string; risk_level: string; status: string; requested_by: string; decided_by?: string; decision_comment?: string; decision_version: number; decided_at?: string; created_at: string }
export interface AgentAttempt { id: string; attempt_number: number; provider: string; model: string; status: string; validation_errors?: string[]; error_message?: string }
export interface ModelUsage { id: string; prompt_name: string; prompt_version: string; schema_version: string; provider: string; model: string; input_tokens: number; output_tokens: number; estimated_cost: string | number; latency_ms: number; status: string }
export interface AnalysisReport { id: string; task_id: string; title: string; summary: string; content_markdown: string; status: string; created_at: string }
export interface TaskDetails { task: AgentTask; attempts: AgentAttempt[]; usage: ModelUsage[]; findings: BusinessFinding[]; recommendations: Recommendation[]; approvals: ApprovalRequest[]; report?: AnalysisReport }
