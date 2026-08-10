ALTER TABLE research_sources
  DROP KEY idx_research_source_schedule,
  DROP COLUMN discovery_schedule_id;

DROP TABLE research_schedule_runs;
DROP TABLE research_schedules;
