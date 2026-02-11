package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const PartnerContextKey = "partner"

// AuthMiddleware authenticates requests using API key
func AuthMiddleware(repos *repository.Repositories, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		apiKey := strings.TrimSpace(parts[1])
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			c.Abort()
			return
		}

		// Note: The current implementation requires that we can look up partners by API key hash.
		// Since bcrypt produces different hashes each time (due to salt), we can't directly lookup.
		// For MVP, we'll need to either:
		// 1. Store a SHA256 hash for lookup + bcrypt for verification (recommended for production)
		// 2. Iterate through all active partners and verify (works but inefficient)
		//
		// For now, the repository's GetByAPIKeyHash should handle this by iterating and verifying.
		// This is a limitation of the current schema - in production, add a lookup_hash column.

		// Look up partner - the repository should handle verification
		partner, err := repos.Partner.GetByAPIKeyHash(c.Request.Context(), apiKey)
		if err != nil {
			logger.Warn("Failed to authenticate partner", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}

		if !partner.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "partner account is inactive"})
			c.Abort()
			return
		}

		// Store partner in context
		c.Set(PartnerContextKey, partner)
		c.Next()
	}
}

// GetPartnerFromContext retrieves the partner from the Gin context
func GetPartnerFromContext(c *gin.Context) (*domain.Partner, bool) {
	partner, exists := c.Get(PartnerContextKey)
	if !exists {
		return nil, false
	}

	p, ok := partner.(*domain.Partner)
	return p, ok
}

// hashAPIKey hashes an API key using bcrypt
func hashAPIKey(apiKey string) string {
	// Use a cost of 10 for API keys (faster than passwords)
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), 10)
	if err != nil {
		// This should never happen, but handle it
		return ""
	}
	return string(hash)
}

// VerifyAPIKey verifies an API key against a hash
func VerifyAPIKey(apiKey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
	return err == nil
}
