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
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	db, err := postgres.NewConnection(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	repos := postgres.NewRepositories(db, logger)

	partners, err := repos.Partner.List(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list partners: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Partners (API keys are only shown when you create a partner):")
	fmt.Println()
	if len(partners) == 0 {
		fmt.Println("  No partners found. Create one with:")
		fmt.Println("  go run cmd/create-partner/main.go \"Partner Name\" \"your-api-key\"")
		os.Exit(0)
	}

	for _, p := range partners {
		active := "active"
		if !p.IsActive {
			active = "inactive"
		}
		fmt.Printf("  ID: %s  Name: %s  (%s)  Created: %s\n",
			p.ID.String(), p.Name, active, p.CreatedAt.Format("2006-01-02 15:04"))
	}
	fmt.Println()
	fmt.Println("Use the API key you saved when creating each partner:")
	fmt.Println("  curl -H \"Authorization: Bearer YOUR_API_KEY\" http://localhost:8081/v1/admin/orders")
}
