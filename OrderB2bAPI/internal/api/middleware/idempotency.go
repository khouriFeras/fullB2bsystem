package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/repository"
)

const IdempotencyKeyHeader = "Idempotency-Key"

// IdempotencyMiddleware handles idempotency key validation
func IdempotencyMiddleware(repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to POST/PUT/PATCH requests
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		idempotencyKey := c.GetHeader(IdempotencyKeyHeader)
		if idempotencyKey == "" {
			c.Next()
			return
		}

		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logger.Error("Failed to read request body for idempotency", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process request"})
			c.Abort()
			return
		}

		// Restore body for handler
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// Calculate request hash
		hash := sha256.Sum256(body)
		requestHash := hex.EncodeToString(hash[:])

		// Check if key exists
		existingKey, err := repos.IdempotencyKey.GetByKey(c.Request.Context(), idempotencyKey)
		if err != nil {
			logger.Error("Failed to check idempotency key", zap.Error(err))
			c.Next()
			return
		}

		if existingKey != nil {
			// Key exists - check if request hash matches
			if existingKey.RequestHash != requestHash {
				// Same key, different payload - conflict
				c.JSON(http.StatusConflict, gin.H{
					"error": "idempotency key conflict: same key used with different payload",
				})
				c.Abort()
				return
			}

			// Same key, same payload - return existing order
			c.Set("idempotency_existing_order_id", existingKey.SupplierOrderID.String())
			c.Set("idempotency_key_used", true)
		} else {
			// New key - will be stored after order creation
			c.Set("idempotency_key", idempotencyKey)
			c.Set("idempotency_request_hash", requestHash)
		}

		c.Next()
	}
}

// GetIdempotencyInfo retrieves idempotency information from context
func GetIdempotencyInfo(c *gin.Context) (key string, requestHash string, existingOrderID string, isExisting bool) {
	if existingID, exists := c.Get("idempotency_existing_order_id"); exists {
		if id, ok := existingID.(string); ok {
			return "", "", id, true
		}
	}

	keyVal, _ := c.Get("idempotency_key")
	hashVal, _ := c.Get("idempotency_request_hash")

	key, _ = keyVal.(string)
	requestHash, _ = hashVal.(string)

	return key, requestHash, "", false
}
