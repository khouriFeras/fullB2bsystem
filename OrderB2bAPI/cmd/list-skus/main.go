package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

// ProductsQuery fetches products with variants
const ProductsQuery = `
query getProducts($first: Int!, $after: String) {
  products(first: $first, after: $after) {
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        id
        title
        variants(first: 250) {
          edges {
            node {
              id
              sku
              title
              price
            }
          }
        }
      }
    }
  }
}
`

type SKUInfo struct {
	SKU         string
	ProductID   int64
	VariantID   int64
	ProductName string
	VariantName string
	Price       string
}

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

	fmt.Println("🔍 Fetching all SKUs from Shopify...")

	// Collect all SKUs
	allSKUs := []SKUInfo{}
	hasNextPage := true
	after := ""

	for hasNextPage {
		variables := map[string]interface{}{
			"first": 50,
		}
		if after != "" {
			variables["after"] = after
		}

		resp, err := client.Execute(ProductsQuery, variables)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query Shopify: %v\n", err)
			os.Exit(1)
		}

		// Parse response
		var result struct {
			Data struct {
				Products struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Edges []struct {
						Node struct {
							ID       string `json:"id"`
							Title    string `json:"title"`
							Variants struct {
								Edges []struct {
									Node struct {
										ID    string `json:"id"`
										SKU   string `json:"sku"`
										Title string `json:"title"`
										Price string `json:"price"`
									} `json:"node"`
								} `json:"edges"`
							} `json:"variants"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"products"`
			} `json:"data"`
		}

		if err := json.Unmarshal(resp.Data, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse response: %v\n", err)
			os.Exit(1)
		}

		// Extract SKUs
		for _, productEdge := range result.Data.Products.Edges {
			product := productEdge.Node
			productID := extractIDFromGID(product.ID)

			for _, variantEdge := range product.Variants.Edges {
				variant := variantEdge.Node

				if variant.SKU != "" {
					variantID := extractIDFromGID(variant.ID)
					allSKUs = append(allSKUs, SKUInfo{
						SKU:         variant.SKU,
						ProductID:   productID,
						VariantID:   variantID,
						ProductName: product.Title,
						VariantName: variant.Title,
						Price:       variant.Price,
					})
				}
			}
		}

		hasNextPage = result.Data.Products.PageInfo.HasNextPage
		after = result.Data.Products.PageInfo.EndCursor

		fmt.Printf("⏳ Fetched %d SKUs so far...\r", len(allSKUs))
	}

	fmt.Printf("\n\n✅ Found %d SKUs with values\n\n", len(allSKUs))

	// Sort by SKU
	sort.Slice(allSKUs, func(i, j int) bool {
		return allSKUs[i].SKU < allSKUs[j].SKU
	})

	// Display first 20 SKUs
	fmt.Println("First 20 SKUs:")
	fmt.Println("─" + strings.Repeat("─", 100))
	for i, sku := range allSKUs {
		if i >= 20 {
			break
		}
		fmt.Printf("SKU: %-20s | Product: %-30s | Price: %s\n",
			sku.SKU, truncate(sku.ProductName, 30), sku.Price)
	}

	if len(allSKUs) > 20 {
		fmt.Printf("\n... and %d more SKUs\n", len(allSKUs)-20)
	}

	// Search for similar SKUs
	fmt.Println("\n🔍 Searching for SKUs containing 'SCM' or '8502':")
	found := false
	for _, sku := range allSKUs {
		if contains(sku.SKU, "SCM") || contains(sku.SKU, "8502") {
			fmt.Printf("\n✅ Found: %s\n", sku.SKU)
			fmt.Printf("   Product: %s\n", sku.ProductName)
			fmt.Printf("   Variant: %s\n", sku.VariantName)
			fmt.Printf("   Price: %s\n", sku.Price)
			fmt.Printf("   Product ID: %d\n", sku.ProductID)
			fmt.Printf("   Variant ID: %d\n", sku.VariantID)
			fmt.Printf("\n   To add: go run cmd/add-sku/main.go \"%s\" %d %d\n",
				sku.SKU, sku.ProductID, sku.VariantID)
			found = true
		}
	}

	if !found {
		fmt.Println("   No SKUs found containing 'SCM' or '8502'")
	}
}

func extractIDFromGID(gid string) int64 {
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
		return 0
	}

	var id int64
	for i := start; i < end; i++ {
		id = id*10 + int64(parts[i]-'0')
	}

	return id
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			strings.Contains(s, substr))))
}
