ALTER TABLE mmp_sync_runs
  ADD COLUMN locked_until DATETIME(3) NULL AFTER started_at,
  ADD COLUMN claim_token CHAR(36) NULL AFTER locked_until,
  ADD KEY idx_mmp_sync_run_lock (status, locked_until);

UPDATE mmp_sync_runs
SET locked_until = DATE_ADD(UTC_TIMESTAMP(3), INTERVAL 2 MINUTE),
    claim_token = UUID()
WHERE status = 'PROCESSING';
