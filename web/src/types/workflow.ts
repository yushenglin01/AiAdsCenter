export interface AgentSpec {
  name: string
  version: string
  description: string
  execution_mode: string
  model?: string
  tools: string[]
  permissions: string[]
  max_steps: number
  timeout: number
  input_schema?: string
  output_schema?: string
}

export interface AgentHealth {
  status: string
  provider: string
  details: string[]
}

export interface AgentDefinition {
  spec: AgentSpec
  availability: string
  details?: string[]
}

export interface AgentCatalogItem {
  definition: AgentDefinition
  health?: AgentHealth
}

export interface AgentTaskSnapshot {
  agent_name: string
  workflow_id: string
  workflow_status: string
  campaign_id: string
  campaign_name: string
  trace_id?: string
  status: string
  execution_mode: string
  started_at?: string
  finished_at?: string
  updated_at: string
}

export interface AgentRuntime {
  agent_name: string
  runtime_status: 'IDLE' | 'QUEUED' | 'RUNNING' | 'FAILED'
  active_tasks: AgentTaskSnapshot[]
  last_task?: AgentTaskSnapshot
}

export interface WorkflowRun {
  workflow_id: string
  workflow_type: string
  game_id: string
  campaign_id: string
  status: string
  current_step: string
  idempotency_key: string
  business_task_id?: string
  error_message?: string
  triggered_by: string
  trace_id?: string
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export interface WorkflowStep {
  step_id: string
  workflow_id: string
  agent_name: string
  sequence_number: number
  status: string
  execution_mode: string
  external_task_id?: string
  output_json?: Record<string, unknown>
  error_message?: string
  started_at?: string
  finished_at?: string
}

export interface WorkflowDetails {
  workflow: WorkflowRun
  steps: WorkflowStep[]
}

export interface WorkflowStartInput {
  game_id: string
  campaign_id: string
  analysis_date?: string
}

export interface WorkflowNotification {
  id: string
  tenant_id: string
  workflow_id: string
  event_type: string
  channel: string
  title: string
  message: string
  status: 'UNREAD' | 'READ'
  metadata?: Record<string, unknown>
  read_by?: string
  read_at?: string
  created_at: string
  updated_at: string
}

export interface OpenClawResult<T> {
  intent: string
  status: string
  data: T
}
