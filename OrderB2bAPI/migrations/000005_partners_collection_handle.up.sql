-- Add collection_handle to partners (per-partner Shopify collection)
ALTER TABLE partners ADD COLUMN IF NOT EXISTS collection_handle VARCHAR(255);

-- Allow NULL for existing partners; new partners should set it via create-partner
COMMENT ON COLUMN partners.collection_handle IS 'Shopify collection handle for this partner catalog (e.g. partner-a-catalog)';
