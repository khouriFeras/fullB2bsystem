package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

type orderService struct {
	repos  *repository.Repositories
	logger *zap.Logger
}

// NewOrderService creates a new order service
func NewOrderService(repos *repository.Repositories, logger *zap.Logger) *orderService {
	return &orderService{
		repos:  repos,
		logger: logger,
	}
}

// CreateOrderFromCart creates a supplier order from a cart submission.
// supplierItems must be partner-scoped (from partner_sku_mappings) so only this partner's SKUs are accepted.
func (s *orderService) CreateOrderFromCart(
	ctx context.Context,
	partnerID uuid.UUID,
	req CartSubmitRequest,
	supplierItems map[string]*domain.PartnerSKUMapping,
) (*domain.SupplierOrder, error) {
	// Build customer name from Zain format: first_name + last_name
	customerName := strings.TrimSpace(req.Customer.FirstName + " " + req.Customer.LastName)
	if customerName == "" {
		customerName = req.Customer.FirstName
	}

	// Create order
	order := &domain.SupplierOrder{
		PartnerID:      partnerID,
		PartnerOrderID: req.PartnerOrderID,
		Status:         domain.OrderStatusIncompleteCaution,
		CustomerName:   customerName,
		CustomerPhone:  req.Customer.Phone,
		CartTotal:      req.Totals.Total,
		PaymentStatus:  "Payment pending",
		PaymentMethod:  req.PaymentMethod,
	}

	// Map Zain shipping fields to internal map (street = address, area = state, country default Jordan)
	country := req.Shipping.Country
	if country == "" {
		country = "Jordan"
	}
	order.ShippingAddress = map[string]interface{}{
		"street":      req.Shipping.Address,
		"city":        req.Shipping.City,
		"postal_code": req.Shipping.PostalCode,
		"country":     country,
	}
	if req.Shipping.Area != "" {
		order.ShippingAddress["state"] = req.Shipping.Area
	}
	if req.Customer.Email != "" {
		order.ShippingAddress["email"] = req.Customer.Email
	}

	// Create order in database
	s.logger.Info("Creating supplier order in database", zap.String("partner_order_id", req.PartnerOrderID))
	if err := s.repos.SupplierOrder.Create(ctx, order); err != nil {
		s.logger.Error("Failed to create supplier order in database", zap.Error(err))
		return nil, err
	}

	// Create order items
	s.logger.Info("Creating order items", zap.Int("item_count", len(req.Items)))
	items := make([]*domain.SupplierOrderItem, 0, len(req.Items))
	for _, cartItem := range req.Items {
		item := &domain.SupplierOrderItem{
			SupplierOrderID: order.ID,
			SKU:             cartItem.SKU,
			Title:           cartItem.Title,
			Price:           cartItem.Price,
			Quantity:        cartItem.Quantity,
			ProductURL:      cartItem.ProductURL,
		}

		if mapping, ok := supplierItems[cartItem.SKU]; ok {
			item.IsSupplierItem = true
			item.ShopifyVariantID = &mapping.ShopifyVariantID
			s.logger.Debug("Item is supplier item", zap.String("sku", cartItem.SKU))
		}

		items = append(items, item)
	}

	// Create items in batch
	s.logger.Info("Inserting order items into database", zap.Int("item_count", len(items)))
	if err := s.repos.SupplierOrderItem.CreateBatch(ctx, items); err != nil {
		s.logger.Error("Failed to create order items in database", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Order items created successfully")

	// Log order creation event
	event := &domain.OrderEvent{
		SupplierOrderID: order.ID,
		EventType:       "order_created",
		EventData: map[string]interface{}{
			"partner_order_id": req.PartnerOrderID,
			"status":           order.Status,
		},
	}
	s.repos.OrderEvent.Create(ctx, event)

	return order, nil
}

// ConfirmOrder confirms an order (idempotent: already confirmed returns success)
func (s *orderService) ConfirmOrder(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.repos.SupplierOrder.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Already confirmed/unfulfilled - idempotent success
	if order.Status == domain.OrderStatusUnfulfilled || order.Status == domain.OrderStatusConfirmed {
		return nil
	}

	// Validate state transition
	if !order.Status.CanTransitionTo(domain.OrderStatusUnfulfilled) {
		return &errors.ErrInvalidStateTransition{
			From: order.Status,
			To:   domain.OrderStatusUnfulfilled,
		}
	}

	// Update status
	if err := s.repos.SupplierOrder.UpdateStatus(ctx, orderID, domain.OrderStatusUnfulfilled, nil); err != nil {
		return err
	}

	// Log event
	event := &domain.OrderEvent{
		SupplierOrderID: orderID,
		EventType:       "status_change",
		EventData: map[string]interface{}{
			"from": order.Status,
			"to":   domain.OrderStatusUnfulfilled,
		},
	}
	s.repos.OrderEvent.Create(ctx, event)

	return nil
}

// RejectOrder rejects an order (idempotent: already rejected returns success)
func (s *orderService) RejectOrder(ctx context.Context, orderID uuid.UUID, reason string) error {
	order, err := s.repos.SupplierOrder.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Already rejected - idempotent success
	if order.Status == domain.OrderStatusRejected {
		return nil
	}

	// Validate state transition
	if !order.Status.CanTransitionTo(domain.OrderStatusRejected) {
		return &errors.ErrInvalidStateTransition{
			From: order.Status,
			To:   domain.OrderStatusRejected,
		}
	}

	// Update status
	if err := s.repos.SupplierOrder.UpdateStatus(ctx, orderID, domain.OrderStatusRejected, &reason); err != nil {
		return err
	}

	// Log event
	event := &domain.OrderEvent{
		SupplierOrderID: orderID,
		EventType:       "status_change",
		EventData: map[string]interface{}{
			"from":   order.Status,
			"to":     domain.OrderStatusRejected,
			"reason": reason,
		},
	}
	s.repos.OrderEvent.Create(ctx, event)

	return nil
}

// ShipOrder marks an order as shipped with tracking information (idempotent: already shipped returns success)
func (s *orderService) ShipOrder(ctx context.Context, orderID uuid.UUID, carrier, trackingNumber string, trackingURL *string) error {
	order, err := s.repos.SupplierOrder.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Already shipped/fulfilled - idempotent success
	if order.Status == domain.OrderStatusFulfilled || order.Status == domain.OrderStatusShipped {
		return nil
	}

	// Validate state transition
	if !order.Status.CanTransitionTo(domain.OrderStatusFulfilled) {
		return &errors.ErrInvalidStateTransition{
			From: order.Status,
			To:   domain.OrderStatusFulfilled,
		}
	}

	// Update tracking
	if err := s.repos.SupplierOrder.UpdateTracking(ctx, orderID, &carrier, &trackingNumber, trackingURL); err != nil {
		return err
	}

	// Log event
	event := &domain.OrderEvent{
		SupplierOrderID: orderID,
		EventType:       "status_change",
		EventData: map[string]interface{}{
			"from":            order.Status,
			"to":              domain.OrderStatusFulfilled,
			"carrier":         carrier,
			"tracking_number": trackingNumber,
		},
	}
	if trackingURL != nil {
		event.EventData["tracking_url"] = *trackingURL
	}
	s.repos.OrderEvent.Create(ctx, event)

	return nil
}
