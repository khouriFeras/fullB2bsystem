package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/api/middleware"
	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/service"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

const deliveryStatusTimeout = 15 * time.Second

// HandleGetOrderDeliveryStatus handles GET /v1/orders/:id/delivery-status
// Partner identifies order by :id (partner_order_id or supplier order UUID).
// We resolve to our order and call GetDeliveryStatus (Wassel) with awb or reference_id (our store lookup).
func HandleGetOrderDeliveryStatus(cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		partner, ok := middleware.GetPartnerFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

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
			logger.Error("Failed to get order for delivery status", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		if order.PartnerID != partner.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		baseURL := cfg.GetDeliveryStatus.BaseURL
		if baseURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "delivery status service is not configured"})
			return
		}

		// Build query: prefer awb (tracking_number); else Shopify order number for Wassel; else resolve from Shopify by partner_order tag; else partner_order_id
		params := url.Values{}
		referenceID := ""
		if order.TrackingNumber != nil && *order.TrackingNumber != "" {
			params.Set("awb", *order.TrackingNumber)
		} else if order.ShopifyOrderID != nil && *order.ShopifyOrderID != "" {
			referenceID = *order.ShopifyOrderID
			params.Set("reference_id", referenceID)
		} else {
			// Fallback: look up Shopify order by partner_order tag so we can query Wassel by order number (e.g. "1034")
			shopifySvc := service.NewShopifyService(cfg.Shopify, repos, logger)
			name, lookupErr := shopifySvc.GetOrderNameByPartnerOrderTag(c.Request.Context(), order.PartnerOrderID)
			if lookupErr != nil {
				logger.Warn("Delivery status: Shopify order lookup by partner_order tag failed, using partner_order_id",
					zap.String("partner_order_id", order.PartnerOrderID), zap.Error(lookupErr))
			}
			if name != "" {
				referenceID = name
				params.Set("reference_id", referenceID)
				logger.Info("Delivery status: resolved Shopify order name for Wassel", zap.String("partner_order_id", order.PartnerOrderID), zap.String("reference_id", referenceID))
				if err := repos.SupplierOrder.UpdateShopifyOrderID(c.Request.Context(), order.ID, referenceID); err == nil {
					order.ShopifyOrderID = &referenceID
				}
			} else {
				params.Set("reference_id", order.PartnerOrderID)
				if lookupErr == nil {
					logger.Debug("Delivery status: no Shopify order found for partner_order tag, using partner_order_id", zap.String("partner_order_id", order.PartnerOrderID))
				}
			}
		}
		params.Set("partner_id", partner.ID.String())

		u, err := url.Parse(baseURL)
		if err != nil {
			logger.Error("Invalid GET_DELIVERY_STATUS_URL", zap.String("url", baseURL), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "delivery status service misconfigured"})
			return
		}
		u.Path = "/shipment"
		u.RawQuery = params.Encode()

		client := &http.Client{Timeout: deliveryStatusTimeout}
		resp, err := client.Get(u.String())
		if err != nil {
			logger.Warn("GetDeliveryStatus request failed", zap.String("url", u.String()), zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "delivery status service unavailable", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Warn("Failed to read GetDeliveryStatus response", zap.Error(err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "delivery status response read failed"})
			return
		}

		if resp.StatusCode != http.StatusOK {
			var errBody map[string]interface{}
			if json.Unmarshal(body, &errBody) == nil {
				c.JSON(http.StatusBadGateway, errBody)
			} else {
				c.JSON(http.StatusBadGateway, gin.H{"error": "delivery status error", "status_code": resp.StatusCode, "body": string(body)})
			}
			return
		}

		var shipmentResult map[string]interface{}
		if err := json.Unmarshal(body, &shipmentResult); err != nil {
			logger.Warn("GetDeliveryStatus returned non-JSON", zap.String("body", string(body)))
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid delivery status response", "body": string(body)})
			return
		}

		// Include order shipping address and partner_id with the Wassel shipment response
		out := gin.H{
			"partner_id":       partner.ID.String(),
			"shipping_address": order.ShippingAddress,
			"shipment":         shipmentResult,
		}
		c.JSON(http.StatusOK, out)

		// Notify partner webhook if configured (fire-and-forget)
		if partner.WebhookURL != nil && *partner.WebhookURL != "" {
			webhookPayload := map[string]interface{}{
				"partner_id":        partner.ID.String(),
				"order_id":         order.ID.String(),
				"partner_order_id": order.PartnerOrderID,
				"shipping_address": order.ShippingAddress,
				"shipment":         shipmentResult,
				"event":            "delivery_status",
			}
			go service.NotifyDeliveryUpdate(*partner.WebhookURL, webhookPayload, logger)
		}
	}
}
