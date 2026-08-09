ALTER TABLE research_sources
  ADD COLUMN discovery_method VARCHAR(30) NOT NULL DEFAULT 'MANUAL' AFTER source_url,
  ADD COLUMN discovery_provider VARCHAR(50) NULL AFTER discovery_method,
  ADD COLUMN discovery_query_hash CHAR(64) NULL AFTER discovery_provider,
  ADD COLUMN discovered_at DATETIME(3) NULL AFTER discovery_query_hash,
  ADD KEY idx_research_source_discovery (tenant_id, discovery_method, discovered_at);
