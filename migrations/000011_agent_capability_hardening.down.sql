ALTER TABLE analysis_reports
  DROP COLUMN generated_at,
  DROP COLUMN provenance_json,
  DROP COLUMN source_digest,
  DROP COLUMN generator_version,
  DROP COLUMN generator_agent;

DROP TABLE IF EXISTS research_sources;
