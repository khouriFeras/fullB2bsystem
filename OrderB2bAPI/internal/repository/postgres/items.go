package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
)

type supplierOrderItemRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewSupplierOrderItemRepository creates a new supplier order item repository
func NewSupplierOrderItemRepository(db *sql.DB, logger *zap.Logger) *supplierOrderItemRepository {
	return &supplierOrderItemRepository{
		db:     db,
		logger: logger,
	}
}

func (r *supplierOrderItemRepository) Create(ctx context.Context, item *domain.SupplierOrderItem) error {
	query := `
		INSERT INTO supplier_order_items (
			id, supplier_order_id, sku, title, price, quantity,
			product_url, is_supplier_item, shopify_variant_id, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}

	_, err := r.db.ExecContext(ctx, query,
		item.ID,
		item.SupplierOrderID,
		item.SKU,
		item.Title,
		item.Price,
		item.Quantity,
		item.ProductURL,
		item.IsSupplierItem,
		item.ShopifyVariantID,
		item.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create supplier order item", zap.Error(err))
		return err
	}

	return nil
}

func (r *supplierOrderItemRepository) CreateBatch(ctx context.Context, items []*domain.SupplierOrderItem) error {
	if len(items) == 0 {
		return nil
	}

	query := `
		INSERT INTO supplier_order_items (
			id, supplier_order_id, sku, title, price, quantity,
			product_url, is_supplier_item, shopify_variant_id, created_at
		)
		VALUES `

	args := make([]interface{}, 0, len(items)*10)
	now := time.Now()

	for i, item := range items {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*10+1, i*10+2, i*10+3, i*10+4, i*10+5, i*10+6, i*10+7, i*10+8, i*10+9, i*10+10)

		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}

		args = append(args,
			item.ID,
			item.SupplierOrderID,
			item.SKU,
			item.Title,
			item.Price,
			item.Quantity,
			item.ProductURL,
			item.IsSupplierItem,
			item.ShopifyVariantID,
			item.CreatedAt,
		)
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to create supplier order items batch", zap.Error(err))
		return err
	}

	return nil
}

func (r *supplierOrderItemRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domain.SupplierOrderItem, error) {
	query := `
		SELECT id, supplier_order_id, sku, title, price, quantity,
			product_url, is_supplier_item, shopify_variant_id, created_at
		FROM supplier_order_items
		WHERE supplier_order_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		r.logger.Error("Failed to get supplier order items by order ID", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var items []*domain.SupplierOrderItem
	for rows.Next() {
		var item domain.SupplierOrderItem
		var productURL sql.NullString
		var shopifyVariantID sql.NullInt64

		err := rows.Scan(
			&item.ID,
			&item.SupplierOrderID,
			&item.SKU,
			&item.Title,
			&item.Price,
			&item.Quantity,
			&productURL,
			&item.IsSupplierItem,
			&shopifyVariantID,
			&item.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		if productURL.Valid {
			item.ProductURL = &productURL.String
		}
		if shopifyVariantID.Valid {
			item.ShopifyVariantID = &shopifyVariantID.Int64
		}

		items = append(items, &item)
	}

	return items, rows.Err()
}
