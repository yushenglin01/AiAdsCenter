import type { AgentTask, AnalysisReport, ApprovalRequest, ModelUsage, Recommendation } from './business'

export interface RecommendationDetail {
  recommendation: Recommendation
  approval?: ApprovalRequest
  campaign_name: string
  task_status: string
  report_id?: string
}

export interface ApprovalDetail {
  approval: ApprovalRequest
  recommendation: Recommendation
  task: AgentTask
  report?: AnalysisReport
  campaign_name: string
}

export interface AuditLog {
  id: string
  actor_id: string
  actor_type: string
  action: string
  resource_type: string
  resource_id: string
  task_id?: string
  before?: Record<string, unknown>
  after?: Record<string, unknown>
  metadata?: Record<string, unknown>
  request_id?: string
  trace_id?: string
  created_at: string
}

export interface ProviderUsageSummary {
  provider: string
  calls: number
  input_tokens: number
  output_tokens: number
  estimated_cost: string | number
  average_latency_ms: number
}

export interface ModelUsageSummary {
  calls: number
  successful_calls: number
  input_tokens: number
  output_tokens: number
  estimated_cost: string | number
  average_latency_ms: number
  success_rate: number
  providers: ProviderUsageSummary[]
}

export interface OperationsSummary {
  pending_approvals: number
  total_tasks: number
  succeeded_tasks: number
  failed_tasks: number
  active_tasks: number
  task_success_rate: number
  model_calls: number
  model_cost: string | number
  audit_events: number
}

export type { ModelUsage }
