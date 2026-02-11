package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

type partnerSKUMappingRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewPartnerSKUMappingRepository creates a new partner SKU mapping repository
func NewPartnerSKUMappingRepository(db *sql.DB, logger *zap.Logger) *partnerSKUMappingRepository {
	return &partnerSKUMappingRepository{db: db, logger: logger}
}

func (r *partnerSKUMappingRepository) GetBySKUAndPartner(ctx context.Context, partnerID uuid.UUID, sku string) (*domain.PartnerSKUMapping, error) {
	query := `
		SELECT id, partner_id, sku, shopify_product_id, shopify_variant_id, title, price, image_url, is_active, created_at, updated_at
		FROM partner_sku_mappings
		WHERE partner_id = $1 AND sku = $2 AND is_active = true
	`
	var m domain.PartnerSKUMapping
	var title, price, imageURL sql.NullString
	err := r.db.QueryRowContext(ctx, query, partnerID, sku).Scan(
		&m.ID, &m.PartnerID, &m.SKU, &m.ShopifyProductID, &m.ShopifyVariantID,
		&title, &price, &imageURL, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, &errors.ErrNotFound{Resource: "partner_sku_mapping", ID: sku}
	}
	if err != nil {
		r.logger.Error("Failed to get partner SKU mapping", zap.Error(err), zap.String("partner_id", partnerID.String()), zap.String("sku", sku))
		return nil, err
	}
	if title.Valid {
		m.Title = &title.String
	}
	if price.Valid {
		m.Price = &price.String
	}
	if imageURL.Valid {
		m.ImageURL = &imageURL.String
	}
	return &m, nil
}

func (r *partnerSKUMappingRepository) ListByPartnerID(ctx context.Context, partnerID uuid.UUID) ([]*domain.PartnerSKUMapping, error) {
	query := `
		SELECT id, partner_id, sku, shopify_product_id, shopify_variant_id, title, price, image_url, is_active, created_at, updated_at
		FROM partner_sku_mappings
		WHERE partner_id = $1 AND is_active = true
		ORDER BY sku ASC
	`
	rows, err := r.db.QueryContext(ctx, query, partnerID)
	if err != nil {
		r.logger.Error("Failed to list partner SKU mappings", zap.Error(err), zap.String("partner_id", partnerID.String()))
		return nil, err
	}
	defer rows.Close()

	var out []*domain.PartnerSKUMapping
	for rows.Next() {
		var m domain.PartnerSKUMapping
		var title, price, imageURL sql.NullString
		err := rows.Scan(
			&m.ID, &m.PartnerID, &m.SKU, &m.ShopifyProductID, &m.ShopifyVariantID,
			&title, &price, &imageURL, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if title.Valid {
			m.Title = &title.String
		}
		if price.Valid {
			m.Price = &price.String
		}
		if imageURL.Valid {
			m.ImageURL = &imageURL.String
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (r *partnerSKUMappingRepository) Upsert(ctx context.Context, m *domain.PartnerSKUMapping) error {
	query := `
		INSERT INTO partner_sku_mappings (id, partner_id, sku, shopify_product_id, shopify_variant_id, title, price, image_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (partner_id, sku) DO UPDATE SET
			shopify_product_id = EXCLUDED.shopify_product_id,
			shopify_variant_id = EXCLUDED.shopify_variant_id,
			title = EXCLUDED.title,
			price = EXCLUDED.price,
			image_url = EXCLUDED.image_url,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.PartnerID, m.SKU, m.ShopifyProductID, m.ShopifyVariantID,
		m.Title, m.Price, m.ImageURL, m.IsActive, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to upsert partner SKU mapping", zap.Error(err), zap.String("partner_id", m.PartnerID.String()), zap.String("sku", m.SKU))
		return err
	}
	return nil
}

func (r *partnerSKUMappingRepository) UpsertBatch(ctx context.Context, partnerID uuid.UUID, mappings []*domain.PartnerSKUMapping) error {
	for _, m := range mappings {
		m.PartnerID = partnerID
		if err := r.Upsert(ctx, m); err != nil {
			return err
		}
	}
	return nil
}
