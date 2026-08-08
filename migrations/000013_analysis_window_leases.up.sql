ALTER TABLE analysis_windows
  ADD COLUMN locked_until DATETIME(3) NULL AFTER next_run_at,
  ADD COLUMN claim_token CHAR(36) NULL AFTER locked_until,
  ADD KEY idx_analysis_window_lock (status, locked_until);

UPDATE analysis_windows
SET locked_until = DATE_ADD(UTC_TIMESTAMP(3), INTERVAL 15 MINUTE)
WHERE status = 'PROCESSING' AND locked_until IS NULL;
