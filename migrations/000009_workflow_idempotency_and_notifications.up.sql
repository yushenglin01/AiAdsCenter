ALTER TABLE workflow_runs
  ADD COLUMN idempotency_key VARCHAR(255) NULL AFTER current_step;

UPDATE workflow_runs
SET idempotency_key = CONCAT('legacy:', id)
WHERE idempotency_key IS NULL;

ALTER TABLE workflow_runs
  MODIFY COLUMN idempotency_key VARCHAR(255) NOT NULL,
  ADD UNIQUE KEY uidx_workflow_run_idempotency (tenant_id, idempotency_key);

CREATE TABLE workflow_notifications (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  workflow_id CHAR(36) NOT NULL,
  event_type VARCHAR(80) NOT NULL,
  channel VARCHAR(30) NOT NULL,
  title VARCHAR(200) NOT NULL,
  message TEXT NOT NULL,
  status VARCHAR(20) NOT NULL,
  metadata_json JSON NULL,
  read_by CHAR(36) NULL,
  read_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_workflow_notification_event (tenant_id, workflow_id, event_type),
  KEY idx_workflow_notifications_tenant_status (tenant_id, status, created_at),
  KEY idx_workflow_notifications_workflow (workflow_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
