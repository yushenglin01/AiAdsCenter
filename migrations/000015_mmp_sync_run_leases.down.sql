ALTER TABLE mmp_sync_runs
  DROP KEY idx_mmp_sync_run_lock,
  DROP COLUMN claim_token,
  DROP COLUMN locked_until;
