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
        status
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

	fmt.Println("🔍 Fetching all products from Shopify...")

	hasNextPage := true
	after := ""
	productCount := 0
	searchTerm := "SCM 8502"

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
							Status   string `json:"status"`
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

		// Search through products
		for _, productEdge := range result.Data.Products.Edges {
			product := productEdge.Node
			productID := extractIDFromGID(product.ID)
			productCount++

			// Check if product title or variant contains search term
			productMatches := containsIgnoreCase(product.Title, searchTerm)
			
			for _, variantEdge := range product.Variants.Edges {
				variant := variantEdge.Node
				variantID := extractIDFromGID(variant.ID)
				
				// Check if SKU, title, or product matches
				skuMatches := variant.SKU != "" && containsIgnoreCase(variant.SKU, searchTerm)
				variantMatches := containsIgnoreCase(variant.Title, searchTerm)
				
				if productMatches || skuMatches || variantMatches {
					fmt.Printf("✅ Found match!\n\n")
					fmt.Printf("Product: %s\n", product.Title)
					fmt.Printf("Status: %s\n", product.Status)
					fmt.Printf("Product ID: %d\n", productID)
					fmt.Printf("\nVariants:\n")
					
					// Show all variants of this product
					for _, v := range product.Variants.Edges {
						vID := extractIDFromGID(v.Node.ID)
						fmt.Printf("  - %s\n", v.Node.Title)
						fmt.Printf("    Variant ID: %d\n", vID)
						if v.Node.SKU != "" {
							fmt.Printf("    SKU: %s\n", v.Node.SKU)
						} else {
							fmt.Printf("    SKU: (not set)\n")
						}
						fmt.Printf("    Price: %s\n", v.Node.Price)
						fmt.Println()
					}
					
					// If we found a matching SKU, show how to add it
					if skuMatches {
						fmt.Printf("To add this SKU mapping:\n")
						fmt.Printf("go run cmd/add-sku/main.go \"%s\" %d %d\n", 
							variant.SKU, productID, variantID)
					} else {
						fmt.Printf("⚠️  Note: This product/variant doesn't have SKU '%s' assigned.\n", searchTerm)
						fmt.Printf("You may need to:\n")
						fmt.Printf("  1. Check if the SKU is set differently in Shopify\n")
						fmt.Printf("  2. Or manually add a variant that matches\n")
					}
				}
			}
		}

		hasNextPage = result.Data.Products.PageInfo.HasNextPage
		after = result.Data.Products.PageInfo.EndCursor
		
		if hasNextPage {
			fmt.Printf("⏳ Searched %d products...\r", productCount)
		}
	}

	fmt.Printf("\n\n✅ Searched %d total products\n", productCount)
	
	if productCount == 0 {
		fmt.Println("\n⚠️  No products found. Check:")
		fmt.Println("  1. Products are published in Shopify")
		fmt.Println("  2. API has 'read_products' permission")
		fmt.Println("  3. You're querying the correct store")
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

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		(strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}
