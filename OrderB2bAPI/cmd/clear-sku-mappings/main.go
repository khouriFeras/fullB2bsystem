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

	// Confirm before deleting
	fmt.Println("⚠️  WARNING: This will delete ALL SKU mappings from the database!")
	fmt.Print("Are you sure you want to continue? (yes/no): ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != "yes" {
		fmt.Println("Operation cancelled.")
		os.Exit(0)
	}

	// Delete all SKU mappings
	ctx := context.Background()
	result, err := db.ExecContext(ctx, "DELETE FROM sku_mappings")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete SKU mappings: %v\n", err)
		os.Exit(1)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get rows affected: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully deleted %d SKU mapping(s) from the database.\n", rowsAffected)
}
