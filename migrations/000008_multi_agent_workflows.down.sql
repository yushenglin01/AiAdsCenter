ALTER TABLE agent_tasks
  DROP KEY idx_agent_tasks_agent_name,
  DROP KEY idx_agent_tasks_parent_task_id,
  DROP KEY idx_agent_tasks_workflow_id,
  DROP COLUMN agent_name,
  DROP COLUMN parent_task_id,
  DROP COLUMN workflow_id;

DROP TABLE IF EXISTS workflow_steps;
DROP TABLE IF EXISTS workflow_runs;
