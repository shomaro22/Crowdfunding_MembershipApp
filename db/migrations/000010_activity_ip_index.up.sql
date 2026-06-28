CREATE INDEX idx_activity_logs_ip_created
  ON activity_logs (ip_address, created_at DESC)
  WHERE ip_address IS NOT NULL;
