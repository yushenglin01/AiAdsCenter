DROP TABLE IF EXISTS analysis_windows;
DROP TABLE IF EXISTS ingestion_messages;

ALTER TABLE data_import_jobs
  DROP KEY idx_import_batch,
  DROP COLUMN collected_at,
  DROP COLUMN period_end,
  DROP COLUMN schema_version,
  DROP COLUMN producer_system,
  DROP COLUMN batch_id;
