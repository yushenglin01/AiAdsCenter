DROP TABLE IF EXISTS workflow_notifications;

ALTER TABLE workflow_runs
  DROP KEY uidx_workflow_run_idempotency,
  DROP COLUMN idempotency_key;
