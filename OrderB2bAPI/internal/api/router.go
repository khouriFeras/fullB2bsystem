package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/api/handlers"
	"github.com/jafarshop/b2bapi/internal/api/middleware"
	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/repository"
)

// NewRouter creates and configures the Gin router
func NewRouter(cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(customRecovery(logger))
	router.Use(loggingMiddleware(logger))

	// Root: friendly response so GET / returns 200 instead of 404
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "B2B Order API",
			"docs":    "https://api.jafarshop.com/health",
			"endpoints": []string{
				"GET /health",
				"POST /internal/webhooks/delivery",
				"GET /v1/catalog/products",
				"POST /v1/carts/submit",
				"GET /v1/orders/:id",
				"GET /v1/orders/:id/delivery-status",
				"GET /v1/admin/orders",
			},
		})
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Internal webhook: GetDeliveryStatus forwards Wassel delivery events here
	router.POST("/internal/webhooks/delivery", handlers.HandleInternalDeliveryWebhook(cfg, repos, logger))

	// Shopify webhook: fulfillment events update order status/tracking in our DB
	router.POST("/webhooks/shopify/fulfillment", handlers.HandleShopifyFulfillmentWebhook(cfg, repos, logger))

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Partner routes (require authentication)
		partnerRoutes := v1.Group("")
		partnerRoutes.Use(middleware.AuthMiddleware(repos, logger))
		partnerRoutes.Use(middleware.IdempotencyMiddleware(repos, logger))
		{
			partnerRoutes.GET("/catalog/products", handlers.HandleGetCatalogProducts(cfg, repos, logger))
			partnerRoutes.POST("/carts/submit", handlers.HandleCartSubmit(cfg, repos, logger))
			partnerRoutes.GET("/orders/:id", handlers.HandleGetOrder(cfg, repos, logger))
			partnerRoutes.GET("/orders/:id/delivery-status", handlers.HandleGetOrderDeliveryStatus(cfg, repos, logger))
		}

		// Partner order listing (check status only - no confirm/reject/ship)
		adminRoutes := v1.Group("/admin")
		adminRoutes.Use(middleware.AuthMiddleware(repos, logger))
		{
			adminRoutes.GET("/orders", handlers.HandleListOrders(repos, logger))
		}
	}

	return router
}

// customRecovery is a custom recovery middleware that logs panics
func customRecovery(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("Panic recovered",
			zap.Any("error", recovered),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal server error",
			"details": fmt.Sprintf("%v", recovered),
		})
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		status := c.Writer.Status()
		logger.Info("HTTP request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
		)
	}
}
