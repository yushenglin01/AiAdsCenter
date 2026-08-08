CREATE TABLE research_sources (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  game_id CHAR(36) NULL,
  campaign_id CHAR(36) NULL,
  category VARCHAR(30) NOT NULL,
  title VARCHAR(300) NOT NULL,
  summary TEXT NOT NULL,
  source_url VARCHAR(1000) NOT NULL,
  publisher VARCHAR(200) NOT NULL,
  published_at DATETIME(3) NOT NULL,
  content_hash CHAR(64) NOT NULL,
  status VARCHAR(20) NOT NULL,
  review_comment TEXT NULL,
  created_by CHAR(36) NOT NULL,
  reviewed_by CHAR(36) NULL,
  reviewed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_research_source_hash (tenant_id, content_hash),
  KEY idx_research_source_scope (tenant_id, game_id, campaign_id),
  KEY idx_research_source_review (tenant_id, status, published_at),
  KEY idx_research_source_category (tenant_id, category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE analysis_reports
  ADD COLUMN generator_agent VARCHAR(80) NULL,
  ADD COLUMN generator_version VARCHAR(40) NULL,
  ADD COLUMN source_digest CHAR(64) NULL,
  ADD COLUMN provenance_json JSON NULL,
  ADD COLUMN generated_at DATETIME(3) NULL;
