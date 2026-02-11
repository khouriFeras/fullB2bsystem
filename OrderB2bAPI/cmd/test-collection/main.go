package mai

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

func main() {
	collectionID := "450553348308"
	if len(os.Args) > 1 {
		collectionID = os.Args[1]
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create Shopify client
	client := shopify.NewClient(cfg.Shopify, logger)
	// Try querying collection by ID
	collectionGID := fmt.Sprintf("gid://shopify/Collection/%s", collectionID)
	
	fmt.Printf("Testing collection query with ID: %s\n", collectionGID)
	fmt.Printf("Numeric ID: %s\n\n", collectionID)

	// Query 1: Try by GID
	query1 := `
	query getCollectionByID($id: ID!) {
		collection(id: $id) {
			id
			title
			handle
		}
	}
	`
	
	fmt.Println("Query 1: By GID")
	variables1 := map[string]interface{}{
		"id": collectionGID,
	}
	
	resp1, err1 := client.Execute(query1, variables1)
	if err1 != nil {
		fmt.Printf("❌ Error: %v\n", err1)
	} else {
		fmt.Printf("✅ Response: %s\n", string(resp1.Data))
		
		var result1 struct {
			Data struct {
				Collection *struct {
					ID     string `json:"id"`
					Title  string `json:"title"`
					Handle string `json:"handle"`
				} `json:"collection"`
			} `json:"data"`
		}
		
		if err := json.Unmarshal(resp1.Data, &result1); err == nil {
			if result1.Data.Collection != nil {
				fmt.Printf("✅ Found collection: %s\n", result1.Data.Collection.Title)
				return
			} else {
				fmt.Printf("⚠️  Collection is null in response\n")
			}
		}
	}

	// Query 2: Try querying products and see if we can find collection info
	fmt.Println("\nQuery 2: Try to find products in any collection")
	query2 := `
	query getProducts($first: Int!) {
		products(first: $first) {
			edges {
				node {
					id
					title
					collections(first: 10) {
						edges {
							node {
								id
								title
							}
						}
					}
				}
			}
		}
	}
	`
	
	variables2 := map[string]interface{}{
		"first": 5,
	}
	
	resp2, err2 := client.Execute(query2, variables2)
	if err2 != nil {
		fmt.Printf("❌ Error: %v\n", err2)
	} else {
		fmt.Printf("✅ Response: %s\n", string(resp2.Data))
	}
	
	// Query 3: Try querying collections with published status
	fmt.Println("\nQuery 3: Try querying published collections")
	query3 := `
	query getCollections($first: Int!) {
		collections(first: $first, query: "published_status:published") {
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
	
	variables3 := map[string]interface{}{
		"first": 10,
	}
	
	resp3, err3 := client.Execute(query3, variables3)
	if err3 != nil {
		fmt.Printf("❌ Error: %v\n", err3)
	} else {
		fmt.Printf("✅ Response: %s\n", string(resp3.Data))
	}
}
