-- Partner-scoped SKU mappings (one catalog per partner)
CREATE TABLE IF NOT EXISTS partner_sku_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    partner_id UUID NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    sku VARCHAR(255) NOT NULL,
    shopify_product_id BIGINT NOT NULL,
    shopify_variant_id BIGINT NOT NULL,
    title VARCHAR(500),
    price VARCHAR(50),
    image_url VARCHAR(500),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(partner_id, sku)
);

CREATE INDEX idx_partner_sku_mappings_partner_id ON partner_sku_mappings(partner_id);
CREATE INDEX idx_partner_sku_mappings_partner_sku ON partner_sku_mappings(partner_id, sku);
CREATE INDEX idx_partner_sku_mappings_is_active ON partner_sku_mappings(is_active);

CREATE TRIGGER update_partner_sku_mappings_updated_at BEFORE UPDATE ON partner_sku_mappings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
