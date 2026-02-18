package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

type supplierOrderRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewSupplierOrderRepository creates a new supplier order repository
func NewSupplierOrderRepository(db *sql.DB, logger *zap.Logger) *supplierOrderRepository {
	return &supplierOrderRepository{
		db:     db,
		logger: logger,
	}
}

func (r *supplierOrderRepository) Create(ctx context.Context, order *domain.SupplierOrder) error {
	query := `
		INSERT INTO supplier_orders (
			id, partner_id, partner_order_id, status, shopify_draft_order_id, shopify_order_id,
			customer_name, customer_phone, shipping_address, cart_total,
			payment_status, payment_method, rejection_reason, tracking_carrier, tracking_number,
			tracking_url, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	now := time.Now()
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = now
	}
	if order.UpdatedAt.IsZero() {
		order.UpdatedAt = now
	}

	shippingAddressJSON, err := json.Marshal(order.ShippingAddress)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		order.ID,
		order.PartnerID,
		order.PartnerOrderID,
		order.Status,
		order.ShopifyDraftOrderID,
		order.ShopifyOrderID,
		order.CustomerName,
		order.CustomerPhone,
		shippingAddressJSON,
		order.CartTotal,
		order.PaymentStatus,
		order.PaymentMethod,
		order.RejectionReason,
		order.TrackingCarrier,
		order.TrackingNumber,
		order.TrackingURL,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create supplier order", zap.Error(err))
		return err
	}

	return nil
}

func (r *supplierOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.SupplierOrder, error) {
	query := `
		SELECT id, partner_id, partner_order_id, status, shopify_draft_order_id, shopify_order_id,
			customer_name, customer_phone, shipping_address, cart_total,
			payment_status, payment_method, rejection_reason, tracking_carrier, tracking_number,
			tracking_url, last_delivery_status, last_delivery_status_label, last_delivery_waybill, last_delivery_image_url, last_delivery_at,
			created_at, updated_at
		FROM supplier_orders
		WHERE id = $1
	`

	var order domain.SupplierOrder
	var shippingAddressJSON []byte
	var shopifyDraftOrderID sql.NullInt64
	var shopifyOrderID sql.NullString
	var customerPhone sql.NullString
	var paymentStatus sql.NullString
	var paymentMethod sql.NullString
	var rejectionReason sql.NullString
	var trackingCarrier sql.NullString
	var trackingNumber sql.NullString
	var trackingURL sql.NullString
	var lastDeliveryStatus sql.NullInt64
	var lastDeliveryStatusLabel sql.NullString
	var lastDeliveryWaybill sql.NullString
	var lastDeliveryImageURL sql.NullString
	var lastDeliveryAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.PartnerID,
		&order.PartnerOrderID,
		&order.Status,
		&shopifyDraftOrderID,
		&shopifyOrderID,
		&order.CustomerName,
		&customerPhone,
		&shippingAddressJSON,
		&order.CartTotal,
		&paymentStatus,
		&paymentMethod,
		&rejectionReason,
		&trackingCarrier,
		&trackingNumber,
		&trackingURL,
		&lastDeliveryStatus,
		&lastDeliveryStatusLabel,
		&lastDeliveryWaybill,
		&lastDeliveryImageURL,
		&lastDeliveryAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errors.ErrNotFound{Resource: "supplier_order", ID: id.String()}
	}
	if err != nil {
		r.logger.Error("Failed to get supplier order by ID", zap.Error(err))
		return nil, err
	}

	if shopifyDraftOrderID.Valid {
		order.ShopifyDraftOrderID = &shopifyDraftOrderID.Int64
	}
	if shopifyOrderID.Valid {
		order.ShopifyOrderID = &shopifyOrderID.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if paymentStatus.Valid {
		order.PaymentStatus = paymentStatus.String
	}
	if paymentMethod.Valid {
		order.PaymentMethod = &paymentMethod.String
	}
	if rejectionReason.Valid {
		order.RejectionReason = &rejectionReason.String
	}
	if trackingCarrier.Valid {
		order.TrackingCarrier = &trackingCarrier.String
	}
	if trackingNumber.Valid {
		order.TrackingNumber = &trackingNumber.String
	}
	if trackingURL.Valid {
		order.TrackingURL = &trackingURL.String
	}
	if lastDeliveryStatus.Valid {
		s := int(lastDeliveryStatus.Int64)
		order.LastDeliveryStatus = &s
	}
	if lastDeliveryStatusLabel.Valid {
		order.LastDeliveryStatusLabel = &lastDeliveryStatusLabel.String
	}
	if lastDeliveryWaybill.Valid {
		order.LastDeliveryWaybill = &lastDeliveryWaybill.String
	}
	if lastDeliveryImageURL.Valid {
		order.LastDeliveryImageURL = &lastDeliveryImageURL.String
	}
	if lastDeliveryAt.Valid {
		order.LastDeliveryAt = &lastDeliveryAt.Time
	}

	if err := json.Unmarshal(shippingAddressJSON, &order.ShippingAddress); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *supplierOrderRepository) GetByPartnerIDAndPartnerOrderID(ctx context.Context, partnerID uuid.UUID, partnerOrderID string) (*domain.SupplierOrder, error) {
	query := `
		SELECT id, partner_id, partner_order_id, status, shopify_draft_order_id, shopify_order_id,
			customer_name, customer_phone, shipping_address, cart_total,
			payment_status, payment_method, rejection_reason, tracking_carrier, tracking_number,
			tracking_url, last_delivery_status, last_delivery_status_label, last_delivery_waybill, last_delivery_image_url, last_delivery_at,
			created_at, updated_at
		FROM supplier_orders
		WHERE partner_id = $1 AND partner_order_id = $2
	`

	var order domain.SupplierOrder
	var shippingAddressJSON []byte
	var shopifyDraftOrderID sql.NullInt64
	var shopifyOrderID sql.NullString
	var customerPhone sql.NullString
	var paymentStatus sql.NullString
	var paymentMethod sql.NullString
	var rejectionReason sql.NullString
	var trackingCarrier sql.NullString
	var trackingNumber sql.NullString
	var trackingURL sql.NullString
	var lastDeliveryStatus sql.NullInt64
	var lastDeliveryStatusLabel sql.NullString
	var lastDeliveryWaybill sql.NullString
	var lastDeliveryImageURL sql.NullString
	var lastDeliveryAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, partnerID, partnerOrderID).Scan(
		&order.ID,
		&order.PartnerID,
		&order.PartnerOrderID,
		&order.Status,
		&shopifyDraftOrderID,
		&shopifyOrderID,
		&order.CustomerName,
		&customerPhone,
		&shippingAddressJSON,
		&order.CartTotal,
		&paymentStatus,
		&paymentMethod,
		&rejectionReason,
		&trackingCarrier,
		&trackingNumber,
		&trackingURL,
		&lastDeliveryStatus,
		&lastDeliveryStatusLabel,
		&lastDeliveryWaybill,
		&lastDeliveryImageURL,
		&lastDeliveryAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errors.ErrNotFound{Resource: "supplier_order", ID: partnerOrderID}
	}
	if err != nil {
		r.logger.Error("Failed to get supplier order by partner ID and order ID", zap.Error(err))
		return nil, err
	}

	if shopifyDraftOrderID.Valid {
		order.ShopifyDraftOrderID = &shopifyDraftOrderID.Int64
	}
	if shopifyOrderID.Valid {
		order.ShopifyOrderID = &shopifyOrderID.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if paymentStatus.Valid {
		order.PaymentStatus = paymentStatus.String
	}
	if paymentMethod.Valid {
		order.PaymentMethod = &paymentMethod.String
	}
	if rejectionReason.Valid {
		order.RejectionReason = &rejectionReason.String
	}
	if trackingCarrier.Valid {
		order.TrackingCarrier = &trackingCarrier.String
	}
	if trackingNumber.Valid {
		order.TrackingNumber = &trackingNumber.String
	}
	if trackingURL.Valid {
		order.TrackingURL = &trackingURL.String
	}
	if lastDeliveryStatus.Valid {
		s := int(lastDeliveryStatus.Int64)
		order.LastDeliveryStatus = &s
	}
	if lastDeliveryStatusLabel.Valid {
		order.LastDeliveryStatusLabel = &lastDeliveryStatusLabel.String
	}
	if lastDeliveryWaybill.Valid {
		order.LastDeliveryWaybill = &lastDeliveryWaybill.String
	}
	if lastDeliveryImageURL.Valid {
		order.LastDeliveryImageURL = &lastDeliveryImageURL.String
	}
	if lastDeliveryAt.Valid {
		order.LastDeliveryAt = &lastDeliveryAt.Time
	}

	if err := json.Unmarshal(shippingAddressJSON, &order.ShippingAddress); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *supplierOrderRepository) GetByPartnerOrderID(ctx context.Context, partnerOrderID string) (*domain.SupplierOrder, error) {
	if partnerOrderID == "" {
		return nil, &errors.ErrNotFound{Resource: "supplier_order", ID: "partner_order_id empty"}
	}
	query := `
		SELECT id, partner_id, partner_order_id, status, shopify_draft_order_id, shopify_order_id,
			customer_name, customer_phone, shipping_address, cart_total,
			payment_status, payment_method, rejection_reason, tracking_carrier, tracking_number,
			tracking_url, created_at, updated_at
		FROM supplier_orders
		WHERE partner_order_id = $1
		LIMIT 1
	`
	var order domain.SupplierOrder
	var shippingAddressJSON []byte
	var shopifyDraftOrderID sql.NullInt64
	var shopifyOrderIDVal sql.NullString
	var customerPhone sql.NullString
	var paymentStatus sql.NullString
	var paymentMethod sql.NullString
	var rejectionReason sql.NullString
	var trackingCarrier sql.NullString
	var trackingNumber sql.NullString
	var trackingURL sql.NullString
	err := r.db.QueryRowContext(ctx, query, partnerOrderID).Scan(
		&order.ID,
		&order.PartnerID,
		&order.PartnerOrderID,
		&order.Status,
		&shopifyDraftOrderID,
		&shopifyOrderIDVal,
		&order.CustomerName,
		&customerPhone,
		&shippingAddressJSON,
		&order.CartTotal,
		&paymentStatus,
		&paymentMethod,
		&rejectionReason,
		&trackingCarrier,
		&trackingNumber,
		&trackingURL,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, &errors.ErrNotFound{Resource: "supplier_order", ID: partnerOrderID}
	}
	if err != nil {
		r.logger.Error("Failed to get supplier order by partner order ID", zap.Error(err), zap.String("partner_order_id", partnerOrderID))
		return nil, err
	}
	if shopifyDraftOrderID.Valid {
		order.ShopifyDraftOrderID = &shopifyDraftOrderID.Int64
	}
	if shopifyOrderIDVal.Valid {
		order.ShopifyOrderID = &shopifyOrderIDVal.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if paymentStatus.Valid {
		order.PaymentStatus = paymentStatus.String
	}
	if paymentMethod.Valid {
		order.PaymentMethod = &paymentMethod.String
	}
	if rejectionReason.Valid {
		order.RejectionReason = &rejectionReason.String
	}
	if trackingCarrier.Valid {
		order.TrackingCarrier = &trackingCarrier.String
	}
	if trackingNumber.Valid {
		order.TrackingNumber = &trackingNumber.String
	}
	if trackingURL.Valid {
		order.TrackingURL = &trackingURL.String
	}
	if err := json.Unmarshal(shippingAddressJSON, &order.ShippingAddress); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *supplierOrderRepository) GetByShopifyOrderID(ctx context.Context, shopifyOrderID string) (*domain.SupplierOrder, error) {
	if shopifyOrderID == "" {
		return nil, &errors.ErrNotFound{Resource: "supplier_order", ID: "shopify_order_id empty"}
	}
	query := `
		SELECT id, partner_id, partner_order_id, status, shopify_draft_order_id, shopify_order_id,
			customer_name, customer_phone, shipping_address, cart_total,
			payment_status, payment_method, rejection_reason, tracking_carrier, tracking_number,
			tracking_url, last_delivery_status, last_delivery_status_label, last_delivery_waybill, last_delivery_image_url, last_delivery_at,
			created_at, updated_at
		FROM supplier_orders
		WHERE shopify_order_id = $1
		LIMIT 1
	`

	var order domain.SupplierOrder
	var shippingAddressJSON []byte
	var shopifyDraftOrderID sql.NullInt64
	var shopifyOrderIDVal sql.NullString
	var customerPhone sql.NullString
	var paymentStatus sql.NullString
	var paymentMethod sql.NullString
	var rejectionReason sql.NullString
	var trackingCarrier sql.NullString
	var trackingNumber sql.NullString
	var trackingURL sql.NullString
	var lastDeliveryStatus sql.NullInt64
	var lastDeliveryStatusLabel sql.NullString
	var lastDeliveryWaybill sql.NullString
	var lastDeliveryImageURL sql.NullString
	var lastDeliveryAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, shopifyOrderID).Scan(
		&order.ID,
		&order.PartnerID,
		&order.PartnerOrderID,
		&order.Status,
		&shopifyDraftOrderID,
		&shopifyOrderIDVal,
		&order.CustomerName,
		&customerPhone,
		&shippingAddressJSON,
		&order.CartTotal,
		&paymentStatus,
		&paymentMethod,
		&rejectionReason,
		&trackingCarrier,
		&trackingNumber,
		&trackingURL,
		&lastDeliveryStatus,
		&lastDeliveryStatusLabel,
		&lastDeliveryWaybill,
		&lastDeliveryImageURL,
		&lastDeliveryAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errors.ErrNotFound{Resource: "supplier_order", ID: shopifyOrderID}
	}
	if err != nil {
		r.logger.Error("Failed to get supplier order by Shopify order ID", zap.Error(err), zap.String("shopify_order_id", shopifyOrderID))
		return nil, err
	}

	if shopifyDraftOrderID.Valid {
		order.ShopifyDraftOrderID = &shopifyDraftOrderID.Int64
	}
	if shopifyOrderIDVal.Valid {
		order.ShopifyOrderID = &shopifyOrderIDVal.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if paymentStatus.Valid {
		order.PaymentStatus = paymentStatus.String
	}
	if paymentMethod.Valid {
		order.PaymentMethod = &paymentMethod.String
	}
	if rejectionReason.Valid {
		order.RejectionReason = &rejectionReason.String
	}
	if trackingCarrier.Valid {
		order.TrackingCarrier = &trackingCarrier.String
	}
	if trackingNumber.Valid {
		order.TrackingNumber = &trackingNumber.String
	}
	if trackingURL.Valid {
		order.TrackingURL = &trackingURL.String
	}
	if lastDeliveryStatus.Valid {
		s := int(lastDeliveryStatus.Int64)
		order.LastDeliveryStatus = &s
	}
	if lastDeliveryStatusLabel.Valid {
		order.LastDeliveryStatusLabel = &lastDeliveryStatusLabel.String
	}
	if lastDeliveryWaybill.Valid {
		order.LastDeliveryWaybill = &lastDeliveryWaybill.String
	}
	if lastDeliveryImageURL.Valid {
		order.LastDeliveryImageURL = &lastDeliveryImageURL.String
	}
	if lastDeliveryAt.Valid {
		order.LastDeliveryAt = &lastDeliveryAt.Time
	}

	if err := json.Unmarshal(shippingAddressJSON, &order.ShippingAddress); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *supplierOrderRepository) Update(ctx context.Context, order *domain.SupplierOrder) error {
	query := `
		UPDATE supplier_orders
		SET status = $2, shopify_draft_order_id = $3, customer_name = $4,
			customer_phone = $5, shipping_address = $6, cart_total = $7,
			payment_status = $8, payment_method = $9, rejection_reason = $10, tracking_carrier = $11,
			tracking_number = $12, tracking_url = $13, updated_at = $14
		WHERE id = $1
	`

	order.UpdatedAt = time.Now()
	shippingAddressJSON, err := json.Marshal(order.ShippingAddress)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		order.ID,
		order.Status,
		order.ShopifyDraftOrderID,
		order.CustomerName,
		order.CustomerPhone,
		shippingAddressJSON,
		order.CartTotal,
		order.PaymentStatus,
		order.PaymentMethod,
		order.RejectionReason,
		order.TrackingCarrier,
		order.TrackingNumber,
		order.TrackingURL,
		order.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update supplier order", zap.Error(err))
		return err
	}

	return nil
}

func (r *supplierOrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus, rejectionReason *string) error {
	query := `
		UPDATE supplier_orders
		SET status = $2, rejection_reason = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, status, rejectionReason, time.Now())
	if err != nil {
		r.logger.Error("Failed to update supplier order status", zap.Error(err))
		return err
	}

	return nil
}

func (r *supplierOrderRepository) UpdateStatusFromShopify(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	query := `
		UPDATE supplier_orders
		SET status = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, time.Now())
	if err != nil {
		r.logger.Error("Failed to update supplier order status from Shopify", zap.Error(err))
		return err
	}
	return nil
}

func (r *supplierOrderRepository) UpdateTracking(ctx context.Context, id uuid.UUID, carrier, trackingNumber, trackingURL *string) error {
	query := `
		UPDATE supplier_orders
		SET tracking_carrier = $2, tracking_number = $3, tracking_url = $4,
			status = $5, updated_at = $6
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, carrier, trackingNumber, trackingURL, domain.OrderStatusFulfilled, time.Now())
	if err != nil {
		r.logger.Error("Failed to update supplier order tracking", zap.Error(err))
		return err
	}

	return nil
}

func (r *supplierOrderRepository) UpdateLastDeliveryStatus(ctx context.Context, id uuid.UUID, status int, statusLabel, waybill, imageURL string) error {
	query := `
		UPDATE supplier_orders
		SET last_delivery_status = $2, last_delivery_status_label = $3, last_delivery_waybill = $4,
			last_delivery_image_url = $5, last_delivery_at = $6, updated_at = $7
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, status, statusLabel, waybill, imageURL, now, now)
	if err != nil {
		r.logger.Error("Failed to update last delivery status", zap.Error(err), zap.String("order_id", id.String()))
		return err
	}
	return nil
}

func (r *supplierOrderRepository) UpdateShopifyDraftOrderID(ctx context.Context, id uuid.UUID, draftOrderID int64) error {
	query := `
		UPDATE supplier_orders
		SET shopify_draft_order_id = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, draftOrderID, time.Now())
	if err != nil {
		r.logger.Error("Failed to update Shopify draft order ID", zap.Error(err))
		return err
	}

	return nil
}

func (r *supplierOrderRepository) UpdateShopifyOrderID(ctx context.Context, id uuid.UUID, orderID string) error {
	query := `
		UPDATE supplier_orders
		SET shopify_order_id = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, orderID, time.Now())
	if err != nil {
		r.logger.Error("Failed to update Shopify order ID", zap.Error(err))
		return err
	}

	return nil
}

func (r *supplierOrderRepository) ListByPartnerID(ctx context.Context, partnerID uuid.UUID, limit, offset int) ([]*domain.SupplierOrder, error) {
	query := `
		SELECT id, partner_id, partner_order_id, status, shopify_draft_order_id, shopify_order_id,
			customer_name, customer_phone, shipping_address, cart_total,
			payment_status, payment_method, rejection_reason, tracking_carrier, tracking_number,
			tracking_url, last_delivery_status, last_delivery_status_label, last_delivery_waybill, last_delivery_image_url, last_delivery_at,
			created_at, updated_at
		FROM supplier_orders
		WHERE partner_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, partnerID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list supplier orders by partner ID", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.SupplierOrder
	for rows.Next() {
		order, err := r.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

func (r *supplierOrderRepository) ListByPartnerIDAndStatus(ctx context.Context, partnerID uuid.UUID, status domain.OrderStatus, limit, offset int) ([]*domain.SupplierOrder, error) {
	query := `
		SELECT id, partner_id, partner_order_id, status, shopify_draft_order_id, shopify_order_id,
			customer_name, customer_phone, shipping_address, cart_total,
			payment_status, payment_method, rejection_reason, tracking_carrier, tracking_number,
			tracking_url, last_delivery_status, last_delivery_status_label, last_delivery_waybill, last_delivery_image_url, last_delivery_at,
			created_at, updated_at
		FROM supplier_orders
		WHERE partner_id = $1 AND status = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, partnerID, status, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list supplier orders by partner ID and status", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.SupplierOrder
	for rows.Next() {
		order, err := r.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

func (r *supplierOrderRepository) ListByStatus(ctx context.Context, status domain.OrderStatus, limit, offset int) ([]*domain.SupplierOrder, error) {
	query := `
		SELECT id, partner_id, partner_order_id, status, shopify_draft_order_id, shopify_order_id,
			customer_name, customer_phone, shipping_address, cart_total,
			payment_status, payment_method, rejection_reason, tracking_carrier, tracking_number,
			tracking_url, last_delivery_status, last_delivery_status_label, last_delivery_waybill, last_delivery_image_url, last_delivery_at,
			created_at, updated_at
		FROM supplier_orders
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list supplier orders by status", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.SupplierOrder
	for rows.Next() {
		order, err := r.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

func (r *supplierOrderRepository) scanOrder(rows *sql.Rows) (*domain.SupplierOrder, error) {
	var order domain.SupplierOrder
	var shippingAddressJSON []byte
	var shopifyDraftOrderID sql.NullInt64
	var shopifyOrderID sql.NullString
	var customerPhone sql.NullString
	var paymentStatus sql.NullString
	var paymentMethod sql.NullString
	var rejectionReason sql.NullString
	var trackingCarrier sql.NullString
	var trackingNumber sql.NullString
	var trackingURL sql.NullString
	var lastDeliveryStatus sql.NullInt64
	var lastDeliveryStatusLabel sql.NullString
	var lastDeliveryWaybill sql.NullString
	var lastDeliveryImageURL sql.NullString
	var lastDeliveryAt sql.NullTime

	err := rows.Scan(
		&order.ID,
		&order.PartnerID,
		&order.PartnerOrderID,
		&order.Status,
		&shopifyDraftOrderID,
		&shopifyOrderID,
		&order.CustomerName,
		&customerPhone,
		&shippingAddressJSON,
		&order.CartTotal,
		&paymentStatus,
		&paymentMethod,
		&rejectionReason,
		&trackingCarrier,
		&trackingNumber,
		&trackingURL,
		&lastDeliveryStatus,
		&lastDeliveryStatusLabel,
		&lastDeliveryWaybill,
		&lastDeliveryImageURL,
		&lastDeliveryAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if shopifyDraftOrderID.Valid {
		order.ShopifyDraftOrderID = &shopifyDraftOrderID.Int64
	}
	if shopifyOrderID.Valid {
		order.ShopifyOrderID = &shopifyOrderID.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if paymentStatus.Valid {
		order.PaymentStatus = paymentStatus.String
	}
	if paymentMethod.Valid {
		order.PaymentMethod = &paymentMethod.String
	}
	if rejectionReason.Valid {
		order.RejectionReason = &rejectionReason.String
	}
	if trackingCarrier.Valid {
		order.TrackingCarrier = &trackingCarrier.String
	}
	if trackingNumber.Valid {
		order.TrackingNumber = &trackingNumber.String
	}
	if trackingURL.Valid {
		order.TrackingURL = &trackingURL.String
	}
	if lastDeliveryStatus.Valid {
		s := int(lastDeliveryStatus.Int64)
		order.LastDeliveryStatus = &s
	}
	if lastDeliveryStatusLabel.Valid {
		order.LastDeliveryStatusLabel = &lastDeliveryStatusLabel.String
	}
	if lastDeliveryWaybill.Valid {
		order.LastDeliveryWaybill = &lastDeliveryWaybill.String
	}
	if lastDeliveryImageURL.Valid {
		order.LastDeliveryImageURL = &lastDeliveryImageURL.String
	}
	if lastDeliveryAt.Valid {
		order.LastDeliveryAt = &lastDeliveryAt.Time
	}

	if err := json.Unmarshal(shippingAddressJSON, &order.ShippingAddress); err != nil {
		return nil, err
	}

	return &order, nil
}
