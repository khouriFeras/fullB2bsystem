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

type skuMappingRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewSKUMappingRepository creates a new SKU mapping repository
func NewSKUMappingRepository(db *sql.DB, logger *zap.Logger) *skuMappingRepository {
	return &skuMappingRepository{
		db:     db,
		logger: logger,
	}
}

func (r *skuMappingRepository) GetBySKU(ctx context.Context, sku string) (*domain.SKUMapping, error) {
	query := `
		SELECT id, sku, shopify_product_id, shopify_variant_id, is_active, created_at, updated_at
		FROM sku_mappings
		WHERE sku = $1
	`

	var mapping domain.SKUMapping

	err := r.db.QueryRowContext(ctx, query, sku).Scan(
		&mapping.ID,
		&mapping.SKU,
		&mapping.ShopifyProductID,
		&mapping.ShopifyVariantID,
		&mapping.IsActive,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errors.ErrNotFound{Resource: "sku_mapping", ID: sku}
	}
	if err != nil {
		r.logger.Error("Failed to get SKU mapping by SKU", zap.Error(err))
		return nil, err
	}

	return &mapping, nil
}

func (r *skuMappingRepository) GetActiveSKUs(ctx context.Context) ([]string, error) {
	query := `
		SELECT sku
		FROM sku_mappings
		WHERE is_active = true
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to get active SKUs", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var skus []string
	for rows.Next() {
		var sku string
		if err := rows.Scan(&sku); err != nil {
			return nil, err
		}
		skus = append(skus, sku)
	}

	return skus, rows.Err()
}

func (r *skuMappingRepository) Create(ctx context.Context, mapping *domain.SKUMapping) error {
	query := `
		INSERT INTO sku_mappings (id, sku, shopify_product_id, shopify_variant_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now()
	if mapping.ID == uuid.Nil {
		mapping.ID = uuid.New()
	}
	if mapping.CreatedAt.IsZero() {
		mapping.CreatedAt = now
	}
	if mapping.UpdatedAt.IsZero() {
		mapping.UpdatedAt = now
	}

	_, err := r.db.ExecContext(ctx, query,
		mapping.ID,
		mapping.SKU,
		mapping.ShopifyProductID,
		mapping.ShopifyVariantID,
		mapping.IsActive,
		mapping.CreatedAt,
		mapping.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create SKU mapping", zap.Error(err))
		return err
	}

	return nil
}

func (r *skuMappingRepository) Update(ctx context.Context, mapping *domain.SKUMapping) error {
	query := `
		UPDATE sku_mappings
		SET shopify_product_id = $2, shopify_variant_id = $3, is_active = $4, updated_at = $5
		WHERE id = $1
	`

	mapping.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		mapping.ID,
		mapping.ShopifyProductID,
		mapping.ShopifyVariantID,
		mapping.IsActive,
		mapping.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update SKU mapping", zap.Error(err))
		return err
	}

	return nil
}

func (r *skuMappingRepository) Upsert(ctx context.Context, mapping *domain.SKUMapping) error {
	query := `
		INSERT INTO sku_mappings (id, sku, shopify_product_id, shopify_variant_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (sku) DO UPDATE SET
			shopify_product_id = EXCLUDED.shopify_product_id,
			shopify_variant_id = EXCLUDED.shopify_variant_id,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	if mapping.ID == uuid.Nil {
		mapping.ID = uuid.New()
	}
	if mapping.CreatedAt.IsZero() {
		mapping.CreatedAt = now
	}
	mapping.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		mapping.ID,
		mapping.SKU,
		mapping.ShopifyProductID,
		mapping.ShopifyVariantID,
		mapping.IsActive,
		mapping.CreatedAt,
		mapping.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to upsert SKU mapping", zap.Error(err))
		return err
	}

	return nil
}

func (r *skuMappingRepository) GetAllActive(ctx context.Context) ([]*domain.SKUMapping, error) {
	query := `
		SELECT id, sku, shopify_product_id, shopify_variant_id, is_active, created_at, updated_at
		FROM sku_mappings
		WHERE is_active = true
		ORDER BY sku ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to get all active SKU mappings", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var mappings []*domain.SKUMapping
	for rows.Next() {
		var mapping domain.SKUMapping
		err := rows.Scan(
			&mapping.ID,
			&mapping.SKU,
			&mapping.ShopifyProductID,
			&mapping.ShopifyVariantID,
			&mapping.IsActive,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		mappings = append(mappings, &mapping)
	}

	return mappings, rows.Err()
}
