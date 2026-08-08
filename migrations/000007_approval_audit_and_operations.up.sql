ALTER TABLE approval_requests
  ADD COLUMN report_id CHAR(36) NULL AFTER task_id,
  ADD COLUMN action VARCHAR(50) NOT NULL DEFAULT '' AFTER report_id,
  ADD COLUMN suggested_value VARCHAR(80) NULL AFTER action,
  ADD COLUMN reason TEXT NULL AFTER suggested_value,
  ADD COLUMN risk_level VARCHAR(20) NOT NULL DEFAULT '' AFTER reason,
  ADD COLUMN decision_version INT NOT NULL DEFAULT 0 AFTER review_comment,
  ADD KEY idx_approval_requests_report_id (report_id),
  ADD KEY idx_approval_requests_status (status),
  ADD KEY idx_approval_requests_requested_by (requested_by);

UPDATE approval_requests a
JOIN recommendations r ON r.id = a.recommendation_id AND r.tenant_id = a.tenant_id
LEFT JOIN analysis_reports p ON p.task_id = a.task_id AND p.tenant_id = a.tenant_id
SET a.report_id = p.id,
    a.action = r.action,
    a.suggested_value = r.suggested_value,
    a.reason = r.description,
    a.risk_level = r.risk_level;

CREATE TABLE audit_logs (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  actor_id CHAR(36) NOT NULL,
  actor_type VARCHAR(30) NOT NULL,
  action VARCHAR(80) NOT NULL,
  resource_type VARCHAR(60) NOT NULL,
  resource_id CHAR(36) NOT NULL,
  task_id CHAR(36) NULL,
  before_json JSON NULL,
  after_json JSON NULL,
  metadata_json JSON NULL,
  request_id VARCHAR(80) NULL,
  trace_id VARCHAR(80) NULL,
  ip_address VARCHAR(80) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_audit_logs_tenant_created (tenant_id, created_at),
  KEY idx_audit_logs_actor (tenant_id, actor_id),
  KEY idx_audit_logs_resource (tenant_id, resource_type, resource_id),
  KEY idx_audit_logs_task (tenant_id, task_id),
  KEY idx_audit_logs_action (tenant_id, action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
