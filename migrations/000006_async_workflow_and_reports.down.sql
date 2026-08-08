DROP TABLE IF EXISTS analysis_reports;
DROP TABLE IF EXISTS task_outbox;

ALTER TABLE agent_tasks
  DROP COLUMN last_heartbeat_at,
  DROP COLUMN queued_at,
  DROP COLUMN queue_retry_count;
