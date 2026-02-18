package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/api/middleware"
	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/service"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

// resolveOrderByIDOrPartnerOrderID fetches an order by UUID or partner_order_id.
// If idParam is a valid UUID, uses GetByID. Otherwise uses GetByPartnerIDAndPartnerOrderID.
func resolveOrderByIDOrPartnerOrderID(ctx context.Context, repos *repository.Repositories, partnerID uuid.UUID, idParam string) (*domain.SupplierOrder, error) {
	orderID, err := uuid.Parse(idParam)
	if err == nil {
		return repos.SupplierOrder.GetByID(ctx, orderID)
	}
	return repos.SupplierOrder.GetByPartnerIDAndPartnerOrderID(ctx, partnerID, idParam)
}

// OrderResponse represents the order response
type OrderResponse struct {
	ID                  string                 `json:"id"`
	PartnerOrderID      string                 `json:"partner_order_id"`
	Status              domain.OrderStatus     `json:"status"`
	ShopifyDraftOrderID *int64                 `json:"shopify_draft_order_id,omitempty"`
	ShopifyOrderID      *string                `json:"shopify_order_id,omitempty"`
	CustomerName        string                 `json:"customer_name"`
	CustomerPhone       string                 `json:"customer_phone,omitempty"`
	ShippingAddress     map[string]interface{} `json:"shipping_address"`
	CartTotal           float64                `json:"cart_total"`
	PaymentStatus       string                 `json:"payment_status,omitempty"`
	PaymentMethod       *string                `json:"payment_method,omitempty"`
	RejectionReason     *string                `json:"rejection_reason,omitempty"`
	TrackingCarrier     *string                `json:"tracking_carrier,omitempty"`
	TrackingNumber      *string                `json:"tracking_number,omitempty"`
	TrackingURL         *string                `json:"tracking_url,omitempty"`
	Items               []OrderItemResponse    `json:"items"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
}

type OrderItemResponse struct {
	SKU              string  `json:"sku"`
	Title            string  `json:"title"`
	Price            float64 `json:"price"`
	Quantity         int     `json:"quantity"`
	ProductURL       *string `json:"product_url,omitempty"`
	IsSupplierItem   bool    `json:"is_supplier_item"`
	ShopifyVariantID *int64  `json:"shopify_variant_id,omitempty"`
	// Enriched from partner catalog (product_title, product_image_url)
	ProductTitle    *string `json:"product_title,omitempty"`
	ProductImageURL *string `json:"product_image_url,omitempty"`
}

// shopifyFulfillmentStatusToOrderStatus maps Shopify displayFulfillmentStatus to our domain.OrderStatus.
func shopifyFulfillmentStatusToOrderStatus(shopifyStatus string) (domain.OrderStatus, bool) {
	switch strings.ToUpper(strings.TrimSpace(shopifyStatus)) {
	case "FULFILLED", "PARTIALLY_FULFILLED", "RESTOCKED":
		return domain.OrderStatusFulfilled, true
	case "UNFULFILLED":
		return domain.OrderStatusUnfulfilled, true
	default:
		return "", false
	}
}

// HandleGetOrder handles GET /v1/orders/:id
func HandleGetOrder(cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get partner from context
		partner, ok := middleware.GetPartnerFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Resolve order by UUID or partner_order_id
		idParam := c.Param("id")
		if idParam == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "order ID or partner_order_id required"})
			return
		}
		order, err := resolveOrderByIDOrPartnerOrderID(c.Request.Context(), repos, partner.ID, idParam)
		if err != nil {
			if _, ok := err.(*errors.ErrNotFound); ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
				return
			}
			logger.Error("Failed to get order", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// Verify partner owns this order
		if order.PartnerID != partner.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		// Sync status from Shopify when we have a linked Shopify order (so status always reflects Shopify)
		if order.ShopifyOrderID != nil && *order.ShopifyOrderID != "" {
			shopifySvc := service.NewShopifyService(cfg.Shopify, repos, logger)
			shopifyStatus, tc, tn, tu, syncErr := shopifySvc.GetOrderFulfillmentStatus(c.Request.Context(), *order.ShopifyOrderID)
			if syncErr != nil {
				logger.Debug("Could not sync order status from Shopify (order may not exist yet)", zap.String("shopify_order_id", *order.ShopifyOrderID), zap.Error(syncErr))
			} else if syncedStatus, ok := shopifyFulfillmentStatusToOrderStatus(shopifyStatus); ok {
				_ = repos.SupplierOrder.UpdateStatusFromShopify(c.Request.Context(), order.ID, syncedStatus)
				order.Status = syncedStatus
				// If Shopify has tracking and we don't, persist it
				if tn != nil && *tn != "" && (order.TrackingNumber == nil || *order.TrackingNumber == "") {
					_ = repos.SupplierOrder.UpdateTracking(c.Request.Context(), order.ID, tc, tn, tu)
					order.TrackingCarrier = tc
					order.TrackingNumber = tn
					order.TrackingURL = tu
				}
			}
		}

		// Get order items
		items, err := repos.SupplierOrderItem.GetByOrderID(c.Request.Context(), order.ID)
		if err != nil {
			logger.Error("Failed to get order items", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// Build response; enrich items with product details from partner catalog
		itemResponses := make([]OrderItemResponse, len(items))
		for i, item := range items {
			resp := OrderItemResponse{
				SKU:              item.SKU,
				Title:            item.Title,
				Price:            item.Price,
				Quantity:         item.Quantity,
				ProductURL:       item.ProductURL,
				IsSupplierItem:   item.IsSupplierItem,
				ShopifyVariantID: item.ShopifyVariantID,
			}
			// Enrich from partner_sku_mappings (product_title, product_image_url)
			if m, err := repos.PartnerSKUMapping.GetBySKUAndPartner(c.Request.Context(), partner.ID, item.SKU); err == nil {
				resp.ProductTitle = m.Title
				resp.ProductImageURL = m.ImageURL
			}
			itemResponses[i] = resp
		}

		response := OrderResponse{
			ID:                  order.ID.String(),
			PartnerOrderID:      order.PartnerOrderID,
			Status:              order.Status,
			ShopifyDraftOrderID: order.ShopifyDraftOrderID,
			ShopifyOrderID:      order.ShopifyOrderID,
			CustomerName:        order.CustomerName,
			ShippingAddress:     order.ShippingAddress,
			CartTotal:           order.CartTotal,
			Items:               itemResponses,
			CreatedAt:           order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:           order.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if order.CustomerPhone != "" {
			response.CustomerPhone = order.CustomerPhone
		}
		if order.PaymentStatus != "" {
			response.PaymentStatus = order.PaymentStatus
		}
		if order.PaymentMethod != nil {
			response.PaymentMethod = order.PaymentMethod
		}
		if order.RejectionReason != nil {
			response.RejectionReason = order.RejectionReason
		}
		if order.TrackingCarrier != nil {
			response.TrackingCarrier = order.TrackingCarrier
		}
		if order.TrackingNumber != nil {
			response.TrackingNumber = order.TrackingNumber
		}
		if order.TrackingURL != nil {
			response.TrackingURL = order.TrackingURL
		}

		c.JSON(http.StatusOK, response)
	}
}
