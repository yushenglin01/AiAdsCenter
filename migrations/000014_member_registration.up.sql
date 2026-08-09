ALTER TABLE users
  ADD COLUMN email VARCHAR(254) NULL AFTER username,
  ADD COLUMN department VARCHAR(120) NOT NULL DEFAULT '' AFTER display_name,
  ADD COLUMN job_title VARCHAR(120) NOT NULL DEFAULT '' AFTER department,
  ADD COLUMN email_verified_at DATETIME(3) NULL AFTER status,
  ADD COLUMN approved_at DATETIME(3) NULL AFTER email_verified_at,
  ADD COLUMN approved_by CHAR(36) NULL AFTER approved_at,
  ADD COLUMN rejected_at DATETIME(3) NULL AFTER approved_by,
  ADD COLUMN rejected_by CHAR(36) NULL AFTER rejected_at,
  ADD COLUMN rejection_reason VARCHAR(500) NOT NULL DEFAULT '' AFTER rejected_by;

UPDATE users
SET email = CONCAT(LOWER(username), '@legacy.local'),
    email_verified_at = CASE WHEN status = 'ACTIVE' THEN created_at ELSE NULL END,
    approved_at = CASE WHEN status = 'ACTIVE' THEN created_at ELSE NULL END
WHERE email IS NULL OR email = '';

ALTER TABLE users
  MODIFY COLUMN email VARCHAR(254) NOT NULL,
  ADD UNIQUE KEY uidx_user_tenant_email (tenant_id, email),
  ADD KEY idx_user_tenant_status_created (tenant_id, status, created_at);

CREATE TABLE IF NOT EXISTS email_verification_tokens (
  id CHAR(36) PRIMARY KEY,
  tenant_id CHAR(36) NOT NULL,
  user_id CHAR(36) NOT NULL,
  token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  consumed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uidx_email_verification_token_hash (token_hash),
  KEY idx_email_verification_user (tenant_id, user_id, consumed_at, created_at),
  KEY idx_email_verification_expiry (expires_at),
  CONSTRAINT fk_email_verification_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_email_verification_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
