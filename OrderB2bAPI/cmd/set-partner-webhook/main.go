package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/repository/postgres"
	"go.uber.org/zap"
)

func main() {
	partnerIDFlag := flag.String("partner-id", "", "Partner UUID (from list-partners)")
	webhookURLFlag := flag.String("webhook-url", "", "Webhook URL to receive delivery updates (empty to clear)")
	flag.Parse()

	partnerIDStr := strings.TrimSpace(*partnerIDFlag)
	webhookURL := strings.TrimSpace(*webhookURLFlag)

	if partnerIDStr == "" {
		fmt.Fprintf(os.Stderr, "Error: --partner-id is required.\n")
		fmt.Fprintf(os.Stderr, "Usage: go run cmd/set-partner-webhook/main.go --partner-id <uuid> --webhook-url <url>\n")
		fmt.Fprintf(os.Stderr, "  Use empty --webhook-url to clear the webhook.\n")
		os.Exit(1)
	}

	partnerID, err := uuid.Parse(partnerIDStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid partner-id UUID: %v\n", err)
		os.Exit(1)
	}

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

	partner, err := repos.Partner.GetByID(context.Background(), partnerID)
	if err != nil || partner == nil {
		fmt.Fprintf(os.Stderr, "Partner not found: %v\n", err)
		os.Exit(1)
	}

	var newURL *string
	if webhookURL != "" {
		newURL = &webhookURL
	}
	partner.WebhookURL = newURL

	if err := repos.Partner.Update(context.Background(), partner); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update partner: %v\n", err)
		os.Exit(1)
	}

	if newURL != nil {
		fmt.Printf("Partner %s webhook_url set to: %s\n", partner.Name, *newURL)
	} else {
		fmt.Printf("Partner %s webhook_url cleared.\n", partner.Name)
	}
}
