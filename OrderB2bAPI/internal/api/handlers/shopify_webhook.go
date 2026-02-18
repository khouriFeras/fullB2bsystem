package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/service"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

type shopifyFulfillmentWebhookBody struct {
	OrderID   int64  `json:"order_id"`
	OrderName string `json:"order_name"`
	Name      string `json:"name"`

	Status string `json:"status"`

	TrackingNumber  string `json:"tracking_number"`
	TrackingCompany string `json:"tracking_company"`
	TrackingURL     string `json:"tracking_url"`
}

func verifyShopifyHMAC(secret string, body []byte, header string) bool {
	if secret == "" || header == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	// constant-time compare
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(header)))
}

// HandleShopifyFulfillmentWebhook handles POST /webhooks/shopify/fulfillment.
// Configure Shopify webhook topics:
// - fulfillments/create
// - fulfillments/update
// This updates supplier_orders.status to FULFILLED and stores tracking info when provided.
func HandleShopifyFulfillmentWebhook(cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := strings.TrimSpace(cfg.ShopifyWebhookSecret)
		if secret == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "shopify webhook not configured"})
			return
		}

		// Read raw body (Shopify HMAC is computed over raw bytes)
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
			return
		}

		hmacHeader := c.GetHeader("X-Shopify-Hmac-Sha256")
		if !verifyShopifyHMAC(secret, bodyBytes, hmacHeader) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
			return
		}

		var body shopifyFulfillmentWebhookBody
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON", "details": err.Error()})
			return
		}

		orderName := strings.TrimSpace(body.OrderName)
		if orderName == "" {
			orderName = strings.TrimSpace(body.Name)
		}
		orderName = strings.TrimPrefix(orderName, "#")

		// If Shopify didn't include order_name/name, resolve via API from numeric order_id
		if orderName == "" && body.OrderID != 0 {
			shopifySvc := service.NewShopifyService(cfg.Shopify, repos, logger)
			if resolvedName, resolveErr := shopifySvc.GetOrderNameByID(c.Request.Context(), body.OrderID); resolveErr == nil {
				orderName = strings.TrimPrefix(strings.TrimSpace(resolvedName), "#")
			} else {
				logger.Warn("Shopify webhook: failed to resolve order name by ID", zap.Int64("order_id", body.OrderID), zap.Error(resolveErr))
			}
		}

		if orderName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "order_name or order_id required"})
			return
		}

		order, err := repos.SupplierOrder.GetByShopifyOrderID(c.Request.Context(), orderName)
		if err != nil {
			if _, ok := err.(*errors.ErrNotFound); ok {
				// Return 200 so Shopify doesn't keep retrying; the order may not exist in our DB.
				c.JSON(http.StatusOK, gin.H{"ok": true, "status": "not_found", "shopify_order_name": orderName})
				return
			}
			logger.Error("Shopify webhook: failed to lookup order", zap.String("shopify_order_name", orderName), zap.Error(err))
			c.JSON(http.StatusOK, gin.H{"ok": true, "status": "error", "message": "order lookup failed"})
			return
		}

		// Mark as fulfilled (webhook is fulfillment-related)
		_ = repos.SupplierOrder.UpdateStatusFromShopify(c.Request.Context(), order.ID, domain.OrderStatusFulfilled)

		// Persist tracking when provided; keep existing fields when not provided
		trackingNumber := strings.TrimSpace(body.TrackingNumber)
		if trackingNumber != "" {
			carrier := order.TrackingCarrier
			if strings.TrimSpace(body.TrackingCompany) != "" {
				s := strings.TrimSpace(body.TrackingCompany)
				carrier = &s
			}
			url := order.TrackingURL
			if strings.TrimSpace(body.TrackingURL) != "" {
				s := strings.TrimSpace(body.TrackingURL)
				url = &s
			}
			num := trackingNumber
			_ = repos.SupplierOrder.UpdateTracking(c.Request.Context(), order.ID, carrier, &num, url)
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":                true,
			"status":            "updated",
			"shopify_order_name": orderName,
			"topic":             c.GetHeader("X-Shopify-Topic"),
		})
	}
}

