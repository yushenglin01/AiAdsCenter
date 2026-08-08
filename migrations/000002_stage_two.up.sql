CREATE TABLE IF NOT EXISTS games (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, code VARCHAR(80) NOT NULL, name VARCHAR(160) NOT NULL,
  package_name VARCHAR(200) NOT NULL DEFAULT '', timezone VARCHAR(64) NOT NULL, currency CHAR(3) NOT NULL, status VARCHAR(20) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3), deleted_at DATETIME(3) NULL,
  UNIQUE KEY uidx_game_tenant_code (tenant_id, code), KEY idx_games_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS channels (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, code VARCHAR(40) NOT NULL, name VARCHAR(80) NOT NULL, provider VARCHAR(40) NOT NULL, status VARCHAR(20) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3), deleted_at DATETIME(3) NULL,
  UNIQUE KEY uidx_channel_tenant_code (tenant_id, code), KEY idx_channels_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS campaigns (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, game_id CHAR(36) NOT NULL, channel_id CHAR(36) NOT NULL,
  external_id VARCHAR(120) NOT NULL, name VARCHAR(160) NOT NULL, country CHAR(2) NOT NULL, daily_budget DECIMAL(20,6) NOT NULL, currency CHAR(3) NOT NULL, status VARCHAR(20) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3), deleted_at DATETIME(3) NULL,
  UNIQUE KEY uidx_campaign_tenant_external (tenant_id, external_id), KEY idx_campaigns_game_id (game_id), KEY idx_campaigns_channel_id (channel_id), KEY idx_campaigns_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS creatives (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, campaign_id CHAR(36) NOT NULL, external_id VARCHAR(120) NOT NULL,
  name VARCHAR(160) NOT NULL, type VARCHAR(30) NOT NULL, status VARCHAR(20) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3), deleted_at DATETIME(3) NULL,
  UNIQUE KEY uidx_creative_tenant_external (tenant_id, external_id), KEY idx_creatives_campaign_id (campaign_id), KEY idx_creatives_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS data_import_jobs (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, game_id CHAR(36) NOT NULL, import_type VARCHAR(40) NOT NULL, source VARCHAR(40) NOT NULL,
  file_name VARCHAR(255) NOT NULL, file_hash CHAR(64) NOT NULL, idempotency_key VARCHAR(255) NOT NULL, period_start DATE NOT NULL, status VARCHAR(20) NOT NULL,
  total_rows INT NOT NULL DEFAULT 0, imported_rows INT NOT NULL DEFAULT 0, skipped_rows INT NOT NULL DEFAULT 0, error_message TEXT NULL,
  created_by CHAR(36) NOT NULL, finished_at DATETIME(3) NULL, created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_import_idempotency (idempotency_key), KEY idx_import_tenant (tenant_id), KEY idx_import_game (game_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS raw_ad_metrics (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, import_job_id CHAR(36) NOT NULL, source VARCHAR(40) NOT NULL, game_id CHAR(36) NOT NULL, campaign_id CHAR(36) NOT NULL,
  date DATE NOT NULL, country CHAR(2) NOT NULL, currency CHAR(3) NOT NULL, spend DECIMAL(20,6) NOT NULL, impressions BIGINT NOT NULL, clicks BIGINT NOT NULL, installs BIGINT NOT NULL, created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_raw_ad_row (tenant_id, source, campaign_id, date, country), KEY idx_raw_ad_import (import_job_id), KEY idx_raw_ad_game (game_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS normalized_ad_metrics LIKE raw_ad_metrics;

CREATE TABLE IF NOT EXISTS mmp_metrics (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, import_job_id CHAR(36) NOT NULL, source VARCHAR(40) NOT NULL, game_id CHAR(36) NOT NULL, campaign_id CHAR(36) NOT NULL,
  date DATE NOT NULL, country CHAR(2) NOT NULL, installs BIGINT NOT NULL, activations BIGINT NOT NULL, payers BIGINT NOT NULL, revenue DECIMAL(20,6) NOT NULL, currency CHAR(3) NOT NULL, created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_mmp_row (tenant_id, source, campaign_id, date, country), KEY idx_mmp_import (import_job_id), KEY idx_mmp_game (game_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS game_revenue_metrics (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, import_job_id CHAR(36) NOT NULL, game_id CHAR(36) NOT NULL, campaign_id CHAR(36) NOT NULL,
  date DATE NOT NULL, country CHAR(2) NOT NULL, registrations BIGINT NOT NULL, active_users BIGINT NOT NULL, payers BIGINT NOT NULL,
  revenue_d1 DECIMAL(20,6) NOT NULL, revenue_d3 DECIMAL(20,6) NOT NULL, revenue_d7 DECIMAL(20,6) NOT NULL, currency CHAR(3) NOT NULL, created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_revenue_row (tenant_id, campaign_id, date, country), KEY idx_revenue_import (import_job_id), KEY idx_revenue_game (game_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS creative_daily_metrics (
  id CHAR(36) PRIMARY KEY, tenant_id CHAR(36) NOT NULL, import_job_id CHAR(36) NOT NULL, game_id CHAR(36) NOT NULL, campaign_id CHAR(36) NOT NULL, creative_id CHAR(36) NOT NULL,
  date DATE NOT NULL, spend DECIMAL(20,6) NOT NULL, impressions BIGINT NOT NULL, clicks BIGINT NOT NULL, installs BIGINT NOT NULL, frequency DECIMAL(12,6) NOT NULL, currency CHAR(3) NOT NULL, created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_creative_metric_row (tenant_id, creative_id, date), KEY idx_creative_metric_import (import_job_id), KEY idx_creative_metric_game (game_id), KEY idx_creative_metric_campaign (campaign_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
