package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository/postgres"
	"go.uber.org/zap"
)

func main() {
	nameFlag := flag.String("name", "", "Partner display name")
	apiKeyFlag := flag.String("api-key", "", "API key for this partner (save it; it cannot be retrieved later)")
	collectionFlag := flag.String("collection", "", "Shopify collection handle for this partner's catalog (e.g. partner-a-catalog)")
	flag.Parse()

	var partnerName, apiKey, collectionHandle string
	if *nameFlag != "" && *apiKeyFlag != "" {
		partnerName = *nameFlag
		apiKey = *apiKeyFlag
		collectionHandle = *collectionFlag
	} else if flag.NArg() >= 2 {
		partnerName = flag.Arg(0)
		apiKey = flag.Arg(1)
		if flag.NArg() >= 3 {
			collectionHandle = flag.Arg(2)
		} else {
			collectionHandle = *collectionFlag
		}
	} else {
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/create-partner/main.go --name \"Partner Name\" --api-key \"your-api-key\" --collection \"partner-catalog-handle\"")
		fmt.Println("  go run cmd/create-partner/main.go \"Partner Name\" \"your-api-key\" [collection-handle]")
		fmt.Println("Example: go run cmd/create-partner/main.go --name \"Zain Shop\" --api-key \"zain-api-key-12345\" --collection \"zain-catalog\"")
		os.Exit(1)
	}
	if collectionHandle == "" {
		fmt.Fprintf(os.Stderr, "Error: --collection (or third argument) is required. Each partner must have a Shopify collection handle for their catalog.\n")
		os.Exit(1)
	}
	// Trim so the stored hash matches what the server receives (AuthMiddleware trims the Bearer token)
	apiKey = strings.TrimSpace(apiKey)
	partnerName = strings.TrimSpace(partnerName)
	collectionHandle = strings.TrimSpace(collectionHandle)
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "Error: API key cannot be empty after trimming.\n")
		os.Exit(1)
	}
	if len(collectionHandle) > 255 {
		fmt.Fprintf(os.Stderr, "Error: collection handle must be at most 255 characters.\n")
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

	// Hash the API key (bcrypt for verification; SHA256 hex for fast lookup)
	apiKeyHash, err := bcrypt.GenerateFromPassword([]byte(apiKey), 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to hash API key: %v\n", err)
		os.Exit(1)
	}
	apiKeyLookup := apiKeyLookupHex(apiKey)

	// Create repositories
	repos := postgres.NewRepositories(db, logger)

	// Create partner
	partner := &domain.Partner{
		Name:             partnerName,
		APIKeyHash:       string(apiKeyHash),
		APIKeyLookup:     apiKeyLookup,
		CollectionHandle: &collectionHandle,
		IsActive:         true,
	}

	err = repos.Partner.Create(context.Background(), partner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create partner: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Partner created successfully!\n\n")
	fmt.Printf("Partner ID: %s\n", partner.ID.String())
	fmt.Printf("Partner Name: %s\n", partner.Name)
	fmt.Printf("Collection Handle: %s\n", collectionHandle)
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Printf("\n⚠️  IMPORTANT: Save this API key securely! You won't be able to see it again.\n")
	fmt.Printf("\nCatalog sync will populate this partner's products from collection '%s' (runs every 10 min).\n", collectionHandle)
	fmt.Printf("\nUse this API key in the Authorization header:\n")
	fmt.Printf("Authorization: Bearer %s\n", apiKey)
}

func apiKeyLookupHex(apiKey string) string {
	h := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(h[:])
}
