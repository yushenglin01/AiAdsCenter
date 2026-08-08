ALTER TABLE model_usage_records
  DROP COLUMN schema_version,
  DROP COLUMN prompt_name;

ALTER TABLE agent_tasks
  DROP COLUMN schema_version,
  DROP COLUMN schema_name;
