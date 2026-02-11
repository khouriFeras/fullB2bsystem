package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository/postgres"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run cmd/add-sku/main.go <sku> <shopify-product-id> <shopify-variant-id>")
		fmt.Println("Example: go run cmd/add-sku/main.go \"PROD-001\" 123456789 987654321")
		os.Exit(1)
	}

	sku := os.Args[1]
	productID, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid product ID: %v\n", err)
		os.Exit(1)
	}

	variantID, err := strconv.ParseInt(os.Args[3], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid variant ID: %v\n", err)
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Connect to database
	db, err := postgres.NewConnection(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create repositories
	repos := postgres.NewRepositories(db, logger)

	// Create SKU mapping
	mapping := &domain.SKUMapping{
		SKU:              sku,
		ShopifyProductID: productID,
		ShopifyVariantID: variantID,
		IsActive:         true,
	}

	err = repos.SKUMapping.Upsert(context.Background(), mapping)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create SKU mapping: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ SKU mapping created successfully!\n\n")
	fmt.Printf("SKU: %s\n", mapping.SKU)
	fmt.Printf("Shopify Product ID: %d\n", mapping.ShopifyProductID)
	fmt.Printf("Shopify Variant ID: %d\n", mapping.ShopifyVariantID)
}
