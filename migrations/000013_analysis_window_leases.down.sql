ALTER TABLE analysis_windows
  DROP KEY idx_analysis_window_lock,
  DROP COLUMN claim_token,
  DROP COLUMN locked_until;
