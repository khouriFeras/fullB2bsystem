DROP INDEX IF EXISTS idx_partners_api_key_lookup;
ALTER TABLE partners DROP COLUMN IF EXISTS api_key_lookup;
