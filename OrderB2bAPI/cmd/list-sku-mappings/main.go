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

	// Connect to database
	db, err := postgres.NewConnection(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create repositories
	repos := postgres.NewRepositories(db, logger)

	// Get all active SKU mappings
	mappings, err := repos.SKUMapping.GetAllActive(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get SKU mappings: %v\n", err)
		os.Exit(1)
	}

	if len(mappings) == 0 {
		fmt.Println("No SKU mappings found in the database.")
		return
	}

	fmt.Printf("📦 Found %d SKU mapping(s) in database:\n\n", len(mappings))
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ %-15s │ %-20s │ %-20s │ %-8s │\n", "SKU", "Product ID", "Variant ID", "Active")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────┤")

	for _, mapping := range mappings {
		active := "Yes"
		if !mapping.IsActive {
			active = "No"
		}
		fmt.Printf("│ %-15s │ %-20d │ %-20d │ %-8s │\n",
			mapping.SKU,
			mapping.ShopifyProductID,
			mapping.ShopifyVariantID,
			active)
	}

	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println("\n💡 To view a specific SKU mapping, you can also query the database directly:")
	fmt.Println("   SELECT * FROM sku_mappings WHERE sku = 'JDTQ1834';")
}
