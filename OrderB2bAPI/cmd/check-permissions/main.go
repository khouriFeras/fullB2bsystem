package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

// Test different queries to see what permissions we have
const TestProductsQuery = `
query {
  products(first: 1) {
    edges {
      node {
        id
        title
      }
    }
  }
}
`

const TestDraftOrdersQuery = `
mutation {
  draftOrderCreate(input: {lineItems: [{title: "Test", price: "1.00", quantity: 1}]}) {
    draftOrder {
      id
    }
    userErrors {
      field
      message
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

	fmt.Println("Checking API permissions...")

	// Test read_products
	fmt.Println("1. Testing 'read_products' permission...")
	resp, err := client.Execute(TestProductsQuery, nil)
	if err != nil {
		fmt.Printf("   ❌ Failed: %v\n", err)
		fmt.Println("   → You need to add 'read_products' scope to your app")
	} else {
		var result struct {
			Data struct {
				Products struct {
					Edges []struct {
						Node struct {
							ID    string `json:"id"`
							Title string `json:"title"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"products"`
			} `json:"data"`
		}
		json.Unmarshal(resp.Data, &result)
		if len(result.Data.Products.Edges) > 0 {
			fmt.Printf("   ✅ Success! Found product: %s\n", result.Data.Products.Edges[0].Node.Title)
		} else {
			fmt.Println("   ✅ Permission works, but no products found")
		}
	}

	// Test write_draft_orders
	fmt.Println("\n2. Testing 'write_draft_orders' permission...")
	resp, err = client.Execute(TestDraftOrdersQuery, nil)
	if err != nil {
		fmt.Printf("   ❌ Failed: %v\n", err)
		fmt.Println("   → You need to add 'write_draft_orders' scope to your app")
	} else {
		var result struct {
			Data struct {
				DraftOrderCreate struct {
					DraftOrder struct {
						ID string `json:"id"`
					} `json:"draftOrder"`
					UserErrors []struct {
						Field   []string `json:"field"`
						Message string   `json:"message"`
					} `json:"userErrors"`
				} `json:"draftOrderCreate"`
			} `json:"data"`
		}
		json.Unmarshal(resp.Data, &result)
		if len(result.Data.DraftOrderCreate.UserErrors) == 0 {
			fmt.Println("   ✅ Success! Can create draft orders")
		} else {
			fmt.Printf("   ⚠️  Permission works, but: %v\n", result.Data.DraftOrderCreate.UserErrors)
		}
	}

	fmt.Println("\n📋 Required scopes for B2B API:")
	fmt.Println("   - read_products (to read products and variants)")
	fmt.Println("   - write_draft_orders (to create draft orders)")
	fmt.Println("\nTo add scopes:")
	fmt.Println("   1. Go to Shopify Admin → Settings → Apps and sales channels")
	fmt.Println("   2. Click 'Develop apps' → Your app")
	fmt.Println("   3. Click 'Configure Admin API scopes'")
	fmt.Println("   4. Add the required scopes")
	fmt.Println("   5. Click 'Save' then 'Install app' (or reinstall)")
}
