ALTER TABLE agent_tasks
  ADD COLUMN queue_retry_count INT NOT NULL DEFAULT 0 AFTER max_attempts,
  ADD COLUMN queued_at DATETIME(3) NULL AFTER scheduled_at,
  ADD COLUMN last_heartbeat_at DATETIME(3) NULL AFTER queued_at;

CREATE TABLE task_outbox (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  task_type VARCHAR(80) NOT NULL,
  payload_json JSON NOT NULL,
  status VARCHAR(30) NOT NULL,
  dispatch_count INT NOT NULL DEFAULT 0,
  next_attempt_at DATETIME(3) NOT NULL,
  last_error TEXT NULL,
  published_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_task_outbox_task (task_id),
  KEY idx_task_outbox_tenant (tenant_id),
  KEY idx_task_outbox_status (status),
  KEY idx_task_outbox_next_attempt (next_attempt_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE analysis_reports (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  title VARCHAR(240) NOT NULL,
  summary TEXT NOT NULL,
  content_markdown LONGTEXT NOT NULL,
  status VARCHAR(30) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_analysis_report_task (task_id),
  KEY idx_analysis_reports_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
