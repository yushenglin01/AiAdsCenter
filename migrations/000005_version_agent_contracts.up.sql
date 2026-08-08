ALTER TABLE agent_tasks
  ADD COLUMN schema_name VARCHAR(80) NOT NULL DEFAULT 'business-agent-output' AFTER task_type,
  ADD COLUMN schema_version VARCHAR(40) NOT NULL DEFAULT '1.0.0' AFTER schema_name;

ALTER TABLE model_usage_records
  ADD COLUMN prompt_name VARCHAR(80) NOT NULL DEFAULT 'business_agent' AFTER task_id,
  ADD COLUMN schema_version VARCHAR(40) NOT NULL DEFAULT '1.0.0' AFTER prompt_version;
