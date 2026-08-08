CREATE TABLE workflow_runs (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  workflow_type VARCHAR(60) NOT NULL,
  game_id CHAR(36) NOT NULL,
  campaign_id CHAR(36) NOT NULL,
  status VARCHAR(30) NOT NULL,
  current_step VARCHAR(80) NOT NULL,
  business_task_id CHAR(36) NULL,
  input_json JSON NULL,
  output_json JSON NULL,
  error_message TEXT NULL,
  triggered_by CHAR(36) NOT NULL,
  trace_id VARCHAR(80) NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_workflow_runs_tenant_created (tenant_id, created_at),
  KEY idx_workflow_runs_status (tenant_id, status),
  KEY idx_workflow_runs_business_task (business_task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE workflow_steps (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  workflow_id CHAR(36) NOT NULL,
  agent_name VARCHAR(80) NOT NULL,
  sequence_number INT NOT NULL,
  status VARCHAR(30) NOT NULL,
  execution_mode VARCHAR(40) NOT NULL,
  external_task_id CHAR(36) NULL,
  input_json JSON NULL,
  output_json JSON NULL,
  error_message TEXT NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_workflow_step_agent (tenant_id, workflow_id, agent_name),
  KEY idx_workflow_steps_workflow_sequence (workflow_id, sequence_number),
  KEY idx_workflow_steps_external_task (external_task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE agent_tasks
  ADD COLUMN workflow_id CHAR(36) NULL AFTER tenant_id,
  ADD COLUMN parent_task_id CHAR(36) NULL AFTER workflow_id,
  ADD COLUMN agent_name VARCHAR(80) NOT NULL DEFAULT 'business-agent' AFTER parent_task_id,
  ADD KEY idx_agent_tasks_workflow_id (workflow_id),
  ADD KEY idx_agent_tasks_parent_task_id (parent_task_id),
  ADD KEY idx_agent_tasks_agent_name (agent_name);
