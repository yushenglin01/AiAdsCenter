ALTER TABLE research_sources
  DROP KEY idx_research_source_discovery,
  DROP COLUMN discovered_at,
  DROP COLUMN discovery_query_hash,
  DROP COLUMN discovery_provider,
  DROP COLUMN discovery_method;
