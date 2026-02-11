package handlers

import (
	"context"
	"net/http"
	"strconv"

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

// CartSubmitRequest represents the cart submission payload
type CartSubmitRequest struct {
	PartnerOrderID string          `json:"partner_order_id" binding:"required"`
	Items          []CartItem      `json:"items" binding:"required,min=1"`
	Customer       CustomerInfo    `json:"customer" binding:"required"`
	Shipping       ShippingAddress `json:"shipping" binding:"required"`
	Totals         CartTotals      `json:"totals" binding:"required"`
}

type CartItem struct {
	SKU        string  `json:"sku" binding:"required"`
	Title      string  `json:"title" binding:"required"`
	Price      float64 `json:"price" binding:"required,min=0"`
	Quantity   int     `json:"quantity" binding:"required,min=1"`
	ProductURL *string `json:"product_url,omitempty"`
}

// CustomerInfo matches Zain format: first_name, last_name, email, phone_number.
type CustomerInfo struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email"`
	Phone     string `json:"phone_number" binding:"required"`
}

// ShippingAddress matches Zain format: city, area, address. Country defaults to Jordan.
type ShippingAddress struct {
	City       string `json:"city" binding:"required"`
	Area       string `json:"area"`
	Address    string `json:"address" binding:"required"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type CartTotals struct {
	Subtotal float64 `json:"subtotal" binding:"required,min=0"`
	Tax      float64 `json:"tax" binding:"min=0"`
	Shipping float64 `json:"shipping" binding:"min=0"`
	Total    float64 `json:"total" binding:"required,min=0"`
}

// CartSubmitResponse represents the response
type CartSubmitResponse struct {
	SupplierOrderID     string             `json:"supplier_order_id"`
	PartnerOrderID      string             `json:"partner_order_id"`
	Status              domain.OrderStatus `json:"status"`
	ShopifyDraftOrderID *int64             `json:"shopify_draft_order_id,omitempty"`
	ShopifyOrderID      *string            `json:"shopify_order_id,omitempty"`
	ShopifyError        string             `json:"shopify_error,omitempty"`
}

func HandleCartSubmit(cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get partner from context
		partner, ok := middleware.GetPartnerFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Check if this is an idempotent request
		_, _, existingOrderID, isExisting := middleware.GetIdempotencyInfo(c)
		if isExisting {
			// Return existing order; sync to Shopify if not yet linked
			orderID, err := uuid.Parse(existingOrderID)
			if err != nil {
				logger.Error("Invalid existing order ID from idempotency", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}

			order, err := repos.SupplierOrder.GetByID(c.Request.Context(), orderID)
			if err != nil {
				logger.Error("Failed to get existing order", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
				return
			}

			resp := buildCartSubmitResponseWithShopifySync(c.Request.Context(), order, partner, repos, cfg, logger)
			c.JSON(http.StatusOK, resp)
			return
		}

		// Parse request - use service types
		var req service.CartSubmitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "validation failed",
				"details": err.Error(),
			})
			return
		}

		// Check for partner catalog SKUs (partner-scoped; only this partner's SKUs accepted)
		logger.Info("Checking cart for partner catalog SKUs", zap.Int("item_count", len(req.Items)), zap.String("partner_id", partner.ID.String()))
		skuService := service.NewSKUService(repos, logger)
		hasPartnerSKU, partnerItems, err := skuService.CheckCartForPartnerSKUs(
			c.Request.Context(),
			partner.ID,
			req.Items,
		)
		if err != nil {
			logger.Error("Failed to check partner SKUs", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		logger.Info("Partner SKU check completed", zap.Bool("has_partner_sku", hasPartnerSKU), zap.Int("partner_item_count", len(partnerItems)))

		if !hasPartnerSKU {
			logger.Info("No partner catalog SKUs found, returning 204")
			c.Status(http.StatusNoContent)
			return
		}

		// Check if order with this partner_order_id already exists (idempotency by partner_order_id)
		existingOrder, err := repos.SupplierOrder.GetByPartnerIDAndPartnerOrderID(
			c.Request.Context(),
			partner.ID,
			req.PartnerOrderID,
		)
		if err == nil && existingOrder != nil {
			// Order already exists - return it; sync to Shopify if not yet linked
			logger.Info("Order already exists, returning existing order",
				zap.String("partner_order_id", req.PartnerOrderID),
				zap.String("order_id", existingOrder.ID.String()))
			resp := buildCartSubmitResponseWithShopifySync(c.Request.Context(), existingOrder, partner, repos, cfg, logger)
			c.JSON(http.StatusOK, resp)
			return
		}
		// If error is "not found", that's expected - continue to create new order
		if _, isNotFound := err.(*errors.ErrNotFound); !isNotFound && err != nil {
			// Real database error - log it but continue (might be transient)
			logger.Warn("Error checking for existing order, will attempt to create",
				zap.Error(err),
				zap.String("partner_order_id", req.PartnerOrderID))
		}

		// Create order (only partner-scoped items are in partnerItems)
		logger.Info("Creating order from cart", zap.String("partner_order_id", req.PartnerOrderID))
		orderService := service.NewOrderService(repos, logger)
		order, err := orderService.CreateOrderFromCart(c.Request.Context(), partner.ID, req, partnerItems)
		if err != nil {
			logger.Error("Failed to create order",
				zap.Error(err),
				zap.String("error_details", err.Error()),
				zap.String("partner_order_id", req.PartnerOrderID))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to create order",
				"details": err.Error(),
			})
			return
		}

		logger.Info("Order created successfully", zap.String("order_id", order.ID.String()))

		// Create Shopify draft order and complete it so it appears in Shopify Orders
		var shopifyErr string
		orderItems, err := repos.SupplierOrderItem.GetByOrderID(c.Request.Context(), order.ID)
		if err != nil {
			logger.Error("Failed to get order items for draft order", zap.Error(err))
			shopifyErr = "get order items: " + err.Error()
		} else {
			shopifyService := service.NewShopifyService(cfg.Shopify, repos, logger)
			draftOrderID, err := shopifyService.CreateDraftOrder(c.Request.Context(), order, orderItems, partner.Name)
			if err != nil {
				logger.Error("Failed to create Shopify draft order", zap.Error(err), zap.String("error_details", err.Error()))
				shopifyErr = "create draft order: " + err.Error()
			} else {
				if err := repos.SupplierOrder.UpdateShopifyDraftOrderID(c.Request.Context(), order.ID, draftOrderID); err != nil {
					logger.Warn("Failed to update order with draft order ID", zap.Error(err))
				}
				order.ShopifyDraftOrderID = &draftOrderID

				shopifyOrderNumericID, err := shopifyService.CompleteDraftOrder(c.Request.Context(), draftOrderID)
				if err != nil {
					logger.Error("Failed to complete Shopify draft order", zap.Error(err))
					shopifyErr = "complete draft order: " + err.Error()
				} else {
					orderName, nameErr := shopifyService.GetOrderNameByID(c.Request.Context(), shopifyOrderNumericID)
					if nameErr != nil {
						logger.Warn("Failed to get Shopify order name, storing numeric ID as string", zap.Error(nameErr))
						orderName = strconv.FormatInt(shopifyOrderNumericID, 10)
					}
					if err := repos.SupplierOrder.UpdateShopifyOrderID(c.Request.Context(), order.ID, orderName); err != nil {
						logger.Warn("Failed to update order with Shopify order ID", zap.Error(err))
					}
					order.ShopifyOrderID = &orderName
					if setErr := shopifyService.SetOrderPartnerMetafield(c.Request.Context(), orderName, partner.Name); setErr != nil {
						logger.Warn("Failed to set order partner metafield", zap.Error(setErr))
					}
				}
			}
		}

		// Build response with Shopify info so caller can see success or error
		resp := CartSubmitResponse{
			SupplierOrderID:     order.ID.String(),
			PartnerOrderID:      order.PartnerOrderID,
			Status:              order.Status,
			ShopifyDraftOrderID: order.ShopifyDraftOrderID,
			ShopifyOrderID:      order.ShopifyOrderID,
			ShopifyError:        shopifyErr,
		}

		// Store idempotency key if provided
		idempotencyKey, requestHash, _, _ := middleware.GetIdempotencyInfo(c)
		if idempotencyKey != "" {
			idempotency := &domain.IdempotencyKey{
				Key:             idempotencyKey,
				PartnerID:       partner.ID,
				SupplierOrderID: order.ID,
				RequestHash:     requestHash,
			}
			if err := repos.IdempotencyKey.Create(c.Request.Context(), idempotency); err != nil {
				logger.Warn("Failed to store idempotency key", zap.Error(err))
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

// buildCartSubmitResponseWithShopifySync builds the cart submit response for an order.
// If the order has no Shopify order linked yet, it attempts to create the draft order and complete it now (retroactive sync).
func buildCartSubmitResponseWithShopifySync(
	ctx context.Context,
	order *domain.SupplierOrder,
	partner *domain.Partner,
	repos *repository.Repositories,
	cfg *config.Config,
	logger *zap.Logger,
) CartSubmitResponse {
	resp := CartSubmitResponse{
		SupplierOrderID:     order.ID.String(),
		PartnerOrderID:      order.PartnerOrderID,
		Status:              order.Status,
		ShopifyDraftOrderID: order.ShopifyDraftOrderID,
		ShopifyOrderID:      order.ShopifyOrderID,
	}
	if order.ShopifyOrderID != nil {
		return resp
	}
	shopifyService := service.NewShopifyService(cfg.Shopify, repos, logger)
	var draftOrderID int64
	if order.ShopifyDraftOrderID != nil {
		draftOrderID = *order.ShopifyDraftOrderID
	} else {
		// Create draft order
		orderItems, getErr := repos.SupplierOrderItem.GetByOrderID(ctx, order.ID)
		if getErr != nil {
			logger.Error("Failed to get order items for Shopify sync", zap.Error(getErr))
			resp.ShopifyError = "get order items: " + getErr.Error()
			return resp
		}
		var createErr error
		draftOrderID, createErr = shopifyService.CreateDraftOrder(ctx, order, orderItems, partner.Name)
		if createErr != nil {
			logger.Error("Failed to create Shopify draft order (retroactive sync)", zap.Error(createErr))
			resp.ShopifyError = "create draft order: " + createErr.Error()
			return resp
		}
		if err := repos.SupplierOrder.UpdateShopifyDraftOrderID(ctx, order.ID, draftOrderID); err != nil {
			logger.Warn("Failed to update order with draft order ID", zap.Error(err))
		}
		order.ShopifyDraftOrderID = &draftOrderID
		resp.ShopifyDraftOrderID = order.ShopifyDraftOrderID
	}
	// Complete draft so it appears in Shopify Orders
	shopifyOrderNumericID, completeErr := shopifyService.CompleteDraftOrder(ctx, draftOrderID)
	if completeErr != nil {
		logger.Error("Failed to complete Shopify draft order (retroactive sync)", zap.Error(completeErr))
		resp.ShopifyError = "complete draft order: " + completeErr.Error()
		return resp
	}
	orderName, nameErr := shopifyService.GetOrderNameByID(ctx, shopifyOrderNumericID)
	if nameErr != nil {
		logger.Warn("Failed to get Shopify order name (retroactive sync), storing numeric ID as string", zap.Error(nameErr))
		orderName = strconv.FormatInt(shopifyOrderNumericID, 10)
	}
	if err := repos.SupplierOrder.UpdateShopifyOrderID(ctx, order.ID, orderName); err != nil {
		logger.Warn("Failed to update order with Shopify order ID", zap.Error(err))
	}
	order.ShopifyOrderID = &orderName
	resp.ShopifyOrderID = order.ShopifyOrderID
	if setErr := shopifyService.SetOrderPartnerMetafield(ctx, orderName, partner.Name); setErr != nil {
		logger.Warn("Failed to set order partner metafield (retroactive sync)", zap.Error(setErr))
	}
	return resp
}
