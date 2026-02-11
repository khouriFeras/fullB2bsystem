package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/api/middleware"
	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/productb2b"
	"github.com/jafarshop/b2bapi/internal/repository"
)

// HandleGetCatalogProducts handles GET /v1/catalog/products (partner's products only)
func HandleGetCatalogProducts(cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		partner, ok := middleware.GetPartnerFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if partner.CollectionHandle == nil || *partner.CollectionHandle == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "partner has no collection_handle; catalog not available",
			})
			return
		}
		collectionHandle := *partner.CollectionHandle
		cursor := c.Query("cursor")
		limit := 25
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n >= 1 && n <= 100 {
				limit = n
			}
		}

		// Prefer ProductB2B when configured
		if cfg.ProductB2B.BaseURL != "" && cfg.ProductB2B.ServiceKey != "" {
			client := productb2b.NewClient(cfg.ProductB2B.BaseURL, cfg.ProductB2B.ServiceKey, logger)
			body, err := client.GetCatalogProducts(c.Request.Context(), collectionHandle, cursor, limit)
			if err == nil {
				logger.Debug("Catalog served from ProductB2B", zap.String("collection_handle", collectionHandle), zap.String("partner_id", partner.ID.String()))
				c.Data(http.StatusOK, "application/json", body)
				return
			}
			logger.Warn("ProductB2B catalog request failed, falling back to partner_sku_mappings", zap.Error(err), zap.String("partner_id", partner.ID.String()), zap.String("collection_handle", collectionHandle))
		} else {
			logger.Debug("ProductB2B not configured, using partner_sku_mappings", zap.Bool("has_base_url", cfg.ProductB2B.BaseURL != ""), zap.Bool("has_service_key", cfg.ProductB2B.ServiceKey != ""))
		}

		// Fallback: return catalog from partner_sku_mappings (last synced data)
		mappings, err := repos.PartnerSKUMapping.ListByPartnerID(c.Request.Context(), partner.ID)
		if err != nil {
			logger.Error("Failed to list partner SKU mappings", zap.Error(err), zap.String("partner_id", partner.ID.String()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		// Build minimal catalog response (list of products/variants from mappings)
		items := make([]map[string]interface{}, 0, len(mappings))
		for _, m := range mappings {
			title, price := "", ""
			if m.Title != nil {
				title = *m.Title
			}
			if m.Price != nil {
				price = *m.Price
			}
			item := map[string]interface{}{
				"sku":       m.SKU,
				"title":     title,
				"price":     price,
				"image_url": m.ImageURL,
			}
			items = append(items, item)
		}
		c.JSON(http.StatusOK, gin.H{
			"data":       items,
			"pagination": gin.H{"hasNextPage": false, "nextCursor": ""},
			"meta":       gin.H{"collection": collectionHandle, "count": len(items)},
		})
	}
}
