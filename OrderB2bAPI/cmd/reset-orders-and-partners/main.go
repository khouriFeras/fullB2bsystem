// reset-orders-and-partners deletes all orders (and related rows) and all partners for a fresh test.
// Run from OrderB2bAPI: go run cmd/reset-orders-and-partners/main.go
// Use same DB as server (e.g. .env or DB_HOST=127.0.0.1 DB_PORT=5434 when DB runs in Docker).
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/repository/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := postgres.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Delete in order (children first)
	tables := []string{
		"idempotency_keys",
		"order_events",
		"supplier_order_items",
		"supplier_orders",
		"partner_sku_mappings",
		"partners",
	}
	for _, table := range tables {
		result, err := db.ExecContext(ctx, "DELETE FROM "+table)
		if err != nil {
			log.Fatalf("Failed to delete from %s: %v", table, err)
		}
		rows, _ := result.RowsAffected()
		fmt.Printf("Deleted %d row(s) from %s\n", rows, table)
	}
	fmt.Println("Done. You can create a new partner with: go run cmd/create-partner/main.go \"Zain Shop\" \"zain-test-api-key-2024\"")
}
