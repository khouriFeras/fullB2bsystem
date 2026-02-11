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
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/find-order/main.go <partner_order_id>")
		fmt.Println("Example: go run cmd/find-order/main.go \"#45246\"")
		os.Exit(1)
	}

	partnerOrderID := os.Args[1]

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

	fmt.Printf("🔍 Searching for order with partner_order_id: %s\n\n", partnerOrderID)

	// Try multiple variations of the partner_order_id
	variations := []string{
		partnerOrderID,
		partnerOrderID[1:], // without #
		"#" + partnerOrderID, // with # if not present
	}

	var id, partnerID, foundPartnerOrderID, status, customerName string
	var cartTotal float64
	var paymentStatus, paymentMethod *string
	var createdAt string
	var found bool

	for _, variation := range variations {
		query := `
			SELECT id, partner_id, partner_order_id, status, customer_name, 
			       cart_total, payment_status, payment_method, created_at
			FROM supplier_orders
			WHERE partner_order_id = $1
			ORDER BY created_at DESC
			LIMIT 1
		`

		err = db.QueryRowContext(context.Background(), query, variation).Scan(
			&id, &partnerID, &foundPartnerOrderID, &status, &customerName,
			&cartTotal, &paymentStatus, &paymentMethod, &createdAt,
		)

		if err == nil {
			found = true
			break
		}
	}

	if !found {
		// List recent orders to help debug
		fmt.Printf("❌ Order not found. Listing recent orders to help debug:\n\n")
		listQuery := `
			SELECT id, partner_order_id, status, customer_name, created_at
			FROM supplier_orders
			ORDER BY created_at DESC
			LIMIT 10
		`
		rows, err := db.QueryContext(context.Background(), listQuery)
		if err == nil {
			defer rows.Close()
			fmt.Printf("Recent orders:\n")
			for rows.Next() {
				var recentID, recentPartnerOrderID, recentStatus, recentCustomerName, recentCreatedAt string
				rows.Scan(&recentID, &recentPartnerOrderID, &recentStatus, &recentCustomerName, &recentCreatedAt)
				fmt.Printf("  - Partner Order ID: %s, Status: %s, Customer: %s, Created: %s\n", 
					recentPartnerOrderID, recentStatus, recentCustomerName, recentCreatedAt)
			}
		}
		os.Exit(1)
	}

	fmt.Printf("✅ Found order!\n\n")
	fmt.Printf("Supplier Order ID (UUID): %s\n", id)
	fmt.Printf("Partner Order ID: %s\n", foundPartnerOrderID)
	fmt.Printf("Partner ID: %s\n", partnerID)
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Customer Name: %s\n", customerName)
	fmt.Printf("Cart Total: %.2f\n", cartTotal)
	if paymentStatus != nil {
		fmt.Printf("Payment Status: %s\n", *paymentStatus)
	}
	if paymentMethod != nil {
		fmt.Printf("Payment Method: %s\n", *paymentMethod)
	}
	fmt.Printf("Created At: %s\n", createdAt)
	fmt.Printf("\nTo get full order details via API:\n")
	fmt.Printf("curl -H \"Authorization: Bearer YOUR_API_KEY\" http://localhost:8080/v1/orders/%s\n", id)
}
