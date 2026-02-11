package main

import (
	"fmt"
	"os"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

// Simple test query
const TestQuery = `
query {
  shop {
    name
    myshopifyDomain
  }
}
`

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Testing Shopify connection...\n\n")
	fmt.Printf("Shop Domain: %s\n", cfg.Shopify.ShopDomain)
	fmt.Printf("Access Token: %s...%s\n", 
		cfg.Shopify.AccessToken[:min(10, len(cfg.Shopify.AccessToken))],
		cfg.Shopify.AccessToken[max(0, len(cfg.Shopify.AccessToken)-4):])
	fmt.Println()

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create Shopify client
	client := shopify.NewClient(cfg.Shopify, logger)

	// Test query
	resp, err := client.Execute(TestQuery, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Connection failed: %v\n\n", err)
		fmt.Println("Please check:")
		fmt.Println("  1. SHOPIFY_SHOP_DOMAIN format: should be 'store-name.myshopify.com' (no https://)")
		fmt.Println("  2. SHOPIFY_ACCESS_TOKEN: should start with 'shpat_' and be the full token")
		fmt.Println("  3. Token permissions: needs 'read_products' scope")
		os.Exit(1)
	}

	fmt.Println("✅ Connection successful!")
	fmt.Printf("Response: %s\n", string(resp.Data))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
