package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/repository/postgres"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/check-order-items/main.go <supplier_order_id>")
		os.Exit(1)
	}

	orderIDStr := os.Args[1]
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid order ID: %v\n", err)
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

	// Get order
	order, err := repos.SupplierOrder.GetByID(context.Background(), orderID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get order: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Order: %s\n", order.PartnerOrderID)
	if order.ShopifyOrderID != nil {
		fmt.Printf("Shopify Order ID: %s\n", *order.ShopifyOrderID)
		fmt.Printf("\nTo verify in Shopify, run:\n")
		fmt.Printf("  go run cmd/get-shopify-order-by-id/main.go %s\n", *order.ShopifyOrderID)
	} else {
		fmt.Printf("Shopify Order ID: (not created yet)\n")
	}
	fmt.Println()

	// Get order items
	items, err := repos.SupplierOrderItem.GetByOrderID(context.Background(), orderID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get order items: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Order Items (%d):\n", len(items))
	fmt.Println("─────────────────────────────────────────────────────────────")
	for i, item := range items {
		fmt.Printf("\nItem %d:\n", i+1)
		fmt.Printf("  SKU: %s\n", item.SKU)
		fmt.Printf("  Title: %s\n", item.Title)
		fmt.Printf("  Price: %.2f\n", item.Price)
		fmt.Printf("  Quantity: %d\n", item.Quantity)
		fmt.Printf("  Is Supplier Item: %v\n", item.IsSupplierItem)
		if item.ShopifyVariantID != nil {
			fmt.Printf("  Shopify Variant ID: %d\n", *item.ShopifyVariantID)
			fmt.Printf("  → This will appear as a VARIANT in Shopify\n")
		} else {
			fmt.Printf("  Shopify Variant ID: (none)\n")
			fmt.Printf("  → This will appear as a CUSTOM LINE ITEM in Shopify\n")
		}
		if item.ProductURL != nil {
			fmt.Printf("  Product URL: %s\n", *item.ProductURL)
		}
	}
	fmt.Println()
}
