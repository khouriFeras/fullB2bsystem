package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

// CollectionsQuery fetches all collections
const CollectionsQuery = `
query getCollections($first: Int!, $after: String, $query: String) {
  collections(first: $first, after: $after, query: $query) {
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        id
        title
        handle
      }
    }
  }
}
`

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

	// Create Shopify client
	client := shopify.NewClient(cfg.Shopify, logger)

	fmt.Println("🔍 Fetching all collections from Shopify...")
	fmt.Println("")

	allCollections := []struct {
		ID     string
		Title  string
		Handle string
	}{}

	hasNextPage := true
	after := ""

	for hasNextPage {
		variables := map[string]interface{}{
			"first": 50,
			"query": "published_status:published",
		}
		if after != "" {
			variables["after"] = after
		}

		resp, err := client.Execute(CollectionsQuery, variables)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query collections: %v\n", err)
			os.Exit(1)
		}

		var result struct {
			Data struct {
				Collections struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node struct {
							ID     string `json:"id"`
							Title  string `json:"title"`
							Handle string `json:"handle"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"collections"`
			} `json:"data"`
		}

		if err := json.Unmarshal(resp.Data, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
			os.Exit(1)
		}

		for _, edge := range result.Data.Collections.Edges {
			allCollections = append(allCollections, struct {
				ID     string
				Title  string
				Handle string
			}{
				ID:     edge.Node.ID,
				Title:  edge.Node.Title,
				Handle: edge.Node.Handle,
			})
		}

		hasNextPage = result.Data.Collections.PageInfo.HasNextPage
		after = result.Data.Collections.PageInfo.EndCursor

		if hasNextPage {
			fmt.Printf("⏳ Fetched %d collections so far...\r", len(allCollections))
		}
	}

	fmt.Printf("\n\n✅ Found %d collection(s)\n\n", len(allCollections))

	if len(allCollections) == 0 {
		fmt.Println("⚠️  No collections found in this store.")
		os.Exit(0)
	}

	// Extract numeric ID from GID for display
	fmt.Println("Collections:")
	fmt.Println(strings.Repeat("─", 80))
	for i, coll := range allCollections {
		numericID := extractNumericID(coll.ID)
		fmt.Printf("%d. Title: %s\n", i+1, coll.Title)
		fmt.Printf("   Handle: %s\n", coll.Handle)
		fmt.Printf("   ID (GID): %s\n", coll.ID)
		if numericID != "" {
			fmt.Printf("   ID (numeric): %s\n", numericID)
		}
		fmt.Println("")
	}

	// Look for "Partner Catalog" or similar
	fmt.Println("Searching for 'Partner Catalog' or similar...")
	found := false
	for _, coll := range allCollections {
		if containsIgnoreCase(coll.Title, "partner") || containsIgnoreCase(coll.Title, "catalog") {
			numericID := extractNumericID(coll.ID)
			fmt.Printf("\n✅ Found: %s\n", coll.Title)
			fmt.Printf("   Handle: %s\n", coll.Handle)
			fmt.Printf("   ID (GID): %s\n", coll.ID)
			if numericID != "" {
				fmt.Printf("   ID (numeric): %s\n", numericID)
				fmt.Printf("\n   To map products from this collection, run:\n")
				fmt.Printf("   go run cmd/map-partner-catalog/main.go %s\n", numericID)
			}
			found = true
		}
	}

	if !found {
		fmt.Println("   No collection found with 'Partner' or 'Catalog' in the title.")
	}
}

func extractNumericID(gid string) string {
	// GID format: gid://shopify/Collection/123456789
	parts := []rune(gid)
	start := -1
	end := len(parts)

	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] >= '0' && parts[i] <= '9' {
			if end == len(parts) {
				end = i + 1
			}
			start = i
		} else if start != -1 {
			break
		}
	}

	if start == -1 {
		return ""
	}

	return string(parts[start:end])
}

func containsIgnoreCase(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	sLower := toLower(s)
	substrLower := toLower(substr)
	return contains(sLower, substrLower)
}

func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
