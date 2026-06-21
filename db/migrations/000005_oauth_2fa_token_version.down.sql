DROP TABLE IF EXISTS totp_backup_codes;
DROP TABLE IF EXISTS oauth_accounts;
ALTER TABLE users DROP COLUMN IF EXISTS token_version;
