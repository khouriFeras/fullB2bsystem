-- Add deterministic lookup for API key (SHA256 hex) so we can find partner in one query
-- then verify with bcrypt. Existing rows have NULL until key is rotated.
ALTER TABLE partners ADD COLUMN IF NOT EXISTS api_key_lookup VARCHAR(64) UNIQUE;
CREATE INDEX IF NOT EXISTS idx_partners_api_key_lookup ON partners(api_key_lookup) WHERE api_key_lookup IS NOT NULL;
