CREATE TABLE IF NOT EXISTS agent_tasks (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  game_id CHAR(36) NOT NULL,
  campaign_id CHAR(36) NOT NULL,
  task_type VARCHAR(50) NOT NULL,
  status VARCHAR(30) NOT NULL,
  priority INT NOT NULL DEFAULT 0,
  idempotency_key VARCHAR(255) NOT NULL,
  input_json JSON NULL,
  output_json JSON NULL,
  error_message TEXT NULL,
  current_step VARCHAR(80) NOT NULL,
  attempt INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 2,
  scheduled_at DATETIME(3) NOT NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  created_by CHAR(36) NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY uidx_agent_task_idempotency (tenant_id, idempotency_key),
  KEY idx_agent_tasks_game_id (game_id),
  KEY idx_agent_tasks_campaign_id (campaign_id),
  KEY idx_agent_tasks_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS agent_task_attempts (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  attempt_number INT NOT NULL,
  provider VARCHAR(50) NOT NULL,
  model VARCHAR(120) NOT NULL,
  status VARCHAR(30) NOT NULL,
  raw_response_json JSON NULL,
  validation_errors JSON NULL,
  error_message TEXT NULL,
  started_at DATETIME(3) NOT NULL,
  finished_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NULL,
  KEY idx_agent_task_attempts_tenant_id (tenant_id),
  KEY idx_agent_task_attempts_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS model_usage_records (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  prompt_version VARCHAR(40) NOT NULL,
  provider VARCHAR(50) NOT NULL,
  model VARCHAR(120) NOT NULL,
  input_tokens INT NOT NULL,
  output_tokens INT NOT NULL,
  estimated_cost DECIMAL(20,8) NOT NULL DEFAULT 0,
  latency_ms BIGINT NOT NULL,
  status VARCHAR(30) NOT NULL,
  created_at DATETIME(3) NULL,
  KEY idx_model_usage_records_tenant_id (tenant_id),
  KEY idx_model_usage_records_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS business_findings (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  game_id CHAR(36) NOT NULL,
  campaign_id CHAR(36) NOT NULL,
  type VARCHAR(60) NOT NULL,
  rule_code VARCHAR(80) NOT NULL,
  severity VARCHAR(20) NOT NULL,
  conclusion VARCHAR(240) NOT NULL,
  description TEXT NOT NULL,
  evidence_json JSON NULL,
  confidence DECIMAL(5,4) NOT NULL,
  created_at DATETIME(3) NULL,
  KEY idx_business_findings_tenant_id (tenant_id),
  KEY idx_business_findings_task_id (task_id),
  KEY idx_business_findings_game_id (game_id),
  KEY idx_business_findings_campaign_id (campaign_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS recommendations (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  campaign_id CHAR(36) NOT NULL,
  action VARCHAR(50) NOT NULL,
  description TEXT NOT NULL,
  priority INT NOT NULL,
  risk_level VARCHAR(20) NOT NULL,
  requires_approval BOOLEAN NOT NULL,
  suggested_value VARCHAR(80) NULL,
  status VARCHAR(30) NOT NULL,
  created_at DATETIME(3) NULL,
  KEY idx_recommendations_tenant_id (tenant_id),
  KEY idx_recommendations_task_id (task_id),
  KEY idx_recommendations_campaign_id (campaign_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS approval_requests (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  recommendation_id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  status VARCHAR(30) NOT NULL,
  requested_by CHAR(36) NOT NULL,
  reviewed_by CHAR(36) NULL,
  review_comment TEXT NULL,
  reviewed_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY uidx_approval_recommendation (recommendation_id),
  KEY idx_approval_requests_tenant_id (tenant_id),
  KEY idx_approval_requests_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
