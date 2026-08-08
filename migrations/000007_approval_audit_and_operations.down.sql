DROP TABLE IF EXISTS audit_logs;

ALTER TABLE approval_requests
  DROP KEY idx_approval_requests_requested_by,
  DROP KEY idx_approval_requests_status,
  DROP KEY idx_approval_requests_report_id,
  DROP COLUMN decision_version,
  DROP COLUMN risk_level,
  DROP COLUMN reason,
  DROP COLUMN suggested_value,
  DROP COLUMN action,
  DROP COLUMN report_id;
