package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port                  string
	Environment           string
	Database              DatabaseConfig
	Shopify               ShopifyConfig
	ProductB2B            ProductB2BConfig
	GetDeliveryStatus     GetDeliveryStatusConfig
	API                   APIConfig
	LogLevel              string
	DeliveryWebhookSecret string // DELIVERY_WEBHOOK_SECRET: auth for POST /internal/webhooks/delivery from GetDeliveryStatus
	ShopifyWebhookSecret  string // SHOPIFY_WEBHOOK_SECRET: verify incoming Shopify webhooks (X-Shopify-Hmac-Sha256)
}

// GetDeliveryStatusConfig is used to call GetDeliveryStatus (Wassel) for shipment/delivery status
type GetDeliveryStatusConfig struct {
	BaseURL string // e.g. http://getdeliverystatus:5000; empty means delivery-status endpoint returns 503
}

// ProductB2BConfig is used to call ProductB2B for catalog and product details
type ProductB2BConfig struct {
	BaseURL    string // e.g. http://productb2b:3000
	ServiceKey string // PRODUCT_B2B_SERVICE_API_KEY
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ShopifyConfig struct {
	ShopDomain  string
	AccessToken string
	APIVersion  string
}

type APIConfig struct {
	KeyHashSalt string
}

func Load() (*Config, error) {
	viper.SetConfigType("env")
	viper.SetConfigName(".env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AddConfigPath("../..")

	// Set defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("LOG_LEVEL", "info")

	// Read from environment variables
	viper.AutomaticEnv()

	// Try to read .env file (optional)
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if .env doesn't exist, we'll use env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	cfg := &Config{
		Port:        getEnvOrViper("PORT", "8080"),
		Environment: getEnvOrViper("ENVIRONMENT", "development"),
		Database: DatabaseConfig{
			Host:     getEnvOrViper("DB_HOST", "localhost"),
			Port:     getEnvOrViper("DB_PORT", "5432"),
			User:     getEnvOrViper("DB_USER", "postgres"),
			Password: getEnvOrViper("DB_PASSWORD", "postgres"),
			DBName:   getEnvOrViper("DB_NAME", "b2bapi"),
			SSLMode:  getEnvOrViper("DB_SSLMODE", "disable"),
		},
		Shopify: ShopifyConfig{
			ShopDomain:  strings.TrimSpace(getEnvOrViper("SHOPIFY_SHOP_DOMAIN", "")),
			AccessToken: strings.TrimSpace(getEnvOrViper("SHOPIFY_ACCESS_TOKEN", "")),
			APIVersion:  getEnvOrViper("SHOPIFY_API_VERSION", "2026-01"),
		},
		ProductB2B: ProductB2BConfig{
			BaseURL:    strings.TrimSpace(getEnvOrViper("PRODUCT_B2B_URL", "")),
			ServiceKey: strings.TrimSpace(getEnvOrViper("PRODUCT_B2B_SERVICE_API_KEY", "")),
		},
		GetDeliveryStatus: GetDeliveryStatusConfig{
			BaseURL: strings.TrimSpace(getEnvOrViper("GET_DELIVERY_STATUS_URL", "")),
		},
		API: APIConfig{
			KeyHashSalt: getEnvOrViper("API_KEY_HASH_SALT", "default-salt-change-in-production"),
		},
		LogLevel:              getEnvOrViper("LOG_LEVEL", "info"),
		DeliveryWebhookSecret: strings.TrimSpace(getEnvOrViper("DELIVERY_WEBHOOK_SECRET", "")),
		ShopifyWebhookSecret:  strings.TrimSpace(getEnvOrViper("SHOPIFY_WEBHOOK_SECRET", "")),
	}

	// Validate required fields
	if cfg.Shopify.ShopDomain == "" {
		return nil, fmt.Errorf("SHOPIFY_SHOP_DOMAIN is required")
	}
	if cfg.Shopify.AccessToken == "" {
		return nil, fmt.Errorf("SHOPIFY_ACCESS_TOKEN is required")
	}

	return cfg, nil
}

func getEnvOrViper(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	if viper.IsSet(key) {
		return viper.GetString(key)
	}
	return defaultValue
}
