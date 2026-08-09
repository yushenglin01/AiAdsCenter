DROP TABLE IF EXISTS email_verification_tokens;

ALTER TABLE users
  DROP KEY idx_user_tenant_status_created,
  DROP KEY uidx_user_tenant_email,
  DROP COLUMN rejection_reason,
  DROP COLUMN rejected_by,
  DROP COLUMN rejected_at,
  DROP COLUMN approved_by,
  DROP COLUMN approved_at,
  DROP COLUMN email_verified_at,
  DROP COLUMN job_title,
  DROP COLUMN department,
  DROP COLUMN email;
