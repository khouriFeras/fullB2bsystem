package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/repository/postgres"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Initialize database
	db, err := postgres.NewConnection(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize repositories
	// Repositories aren't needed here since we query via SQL directly.
	_ = postgres.NewRepositories(db, logger)

	fmt.Println("📋 Listing all orders in database:")

	// List all orders
	query := `
		SELECT id, partner_id, partner_order_id, status, customer_name,
		       cart_total, payment_status, payment_method, shopify_draft_order_id, shopify_order_id, created_at
		FROM supplier_orders
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query orders: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var id, partnerID, partnerOrderID, status, customerName string
		var cartTotal float64
		var paymentStatus, paymentMethod *string
		var shopifyDraftOrderID *int64
		var shopifyOrderID *string
		var createdAt string

		err := rows.Scan(&id, &partnerID, &partnerOrderID, &status, &customerName,
			&cartTotal, &paymentStatus, &paymentMethod, &shopifyDraftOrderID, &shopifyOrderID, &createdAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to scan row: %v\n", err)
			continue
		}

		count++
		fmt.Printf("Order #%d:\n", count)
		fmt.Printf("  Supplier Order ID: %s\n", id)
		fmt.Printf("  Partner Order ID: %s\n", partnerOrderID)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Customer: %s\n", customerName)
		fmt.Printf("  Total: %.2f\n", cartTotal)
		if paymentStatus != nil {
			fmt.Printf("  Payment Status: %s\n", *paymentStatus)
		}
		if paymentMethod != nil {
			fmt.Printf("  Payment Method: %s\n", *paymentMethod)
		}
		if shopifyDraftOrderID != nil {
			fmt.Printf("  Shopify Draft Order ID: %d\n", *shopifyDraftOrderID)
		}
		if shopifyOrderID != nil {
			fmt.Printf("  Shopify Order ID: %s\n", *shopifyOrderID)
		}
		fmt.Printf("  Created: %s\n", createdAt)
		fmt.Println()
	}

	if count == 0 {
		fmt.Println("❌ No orders found in database.")
		fmt.Println("\nTo test the API, you need to:")
		fmt.Println("1. Create a partner: go run cmd/create-partner/main.go \"Test Partner\" \"test-api-key\"")
		fmt.Println("2. Submit a cart via POST /v1/carts/submit")
		fmt.Println("3. Then query the order status")
	} else {
		fmt.Printf("✅ Found %d order(s)\n", count)
	}
}
