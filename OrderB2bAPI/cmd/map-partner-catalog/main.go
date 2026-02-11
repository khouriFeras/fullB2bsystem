package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository/postgres"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

// CollectionQuery finds a collection by title
const CollectionQuery = `
query getCollectionByTitle($first: Int!, $query: String!) {
  collections(first: $first, query: $query) {
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

// ProductsByCollectionQuery fetches products from a collection
const ProductsByCollectionQuery = `
query getProductsByCollection($collectionId: ID!, $first: Int!, $after: String) {
  collection(id: $collectionId) {
    id
    title
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
}
`

type ProductInfo struct {
	ProductID   int64
	VariantID   int64
	SKU         string
	ProductName string
	VariantName string
	Price       string
}

func main() {
	var collectionID string
	var collectionTitle string
	var partnerName string // if set, write to partner_sku_mappings for this partner

	if len(os.Args) > 1 {
		// Check if argument is a number (collection ID) or text (collection title)
		if isNumeric(os.Args[1]) {
			collectionID = fmt.Sprintf("gid://shopify/Collection/%s", os.Args[1])
		} else {
			collectionTitle = os.Args[1]
		}
	}
	if len(os.Args) > 2 {
		partnerName = strings.TrimSpace(os.Args[2])
	}
	if collectionTitle == "" && collectionID == "" {
		collectionTitle = "Partner Catalog"
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

	// Create Shopify client
	client := shopify.NewClient(cfg.Shopify, logger)

	var collection struct {
		ID     string
		Title  string
		Handle string
	}

	if collectionID != "" {
		// Use collection ID directly
		fmt.Printf("🔍 Using collection ID: %s\n\n", collectionID)

		// Query collection by ID to get title
		collectionByIDQuery := `
		query getCollectionByID($id: ID!) {
			collection(id: $id) {
				id
				title
				handle
			}
		}
		`

		variables := map[string]interface{}{
			"id": collectionID,
		}

		resp, err := client.Execute(collectionByIDQuery, variables)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query collection by ID: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nTrying to list all collections to find the correct ID...\n")
			fmt.Fprintf(os.Stderr, "Run: go run cmd/list-collections/main.go\n")
			os.Exit(1)
		}

		var result struct {
			Collection *struct {
				ID     string `json:"id"`
				Title  string `json:"title"`
				Handle string `json:"handle"`
			} `json:"collection"`
		}

		if err := json.Unmarshal(resp.Data, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse collection response: %v\n", err)
			fmt.Fprintf(os.Stderr, "Raw response: %s\n", string(resp.Data))
			os.Exit(1)
		}

		if result.Collection == nil || result.Collection.ID == "" {
			fmt.Fprintf(os.Stderr, "❌ Collection with ID '%s' not found!\n", collectionID)
			fmt.Fprintf(os.Stderr, "\nPossible reasons:\n")
			fmt.Fprintf(os.Stderr, "  1. Collection ID might be from a different store\n")
			fmt.Fprintf(os.Stderr, "  2. Collection might not exist\n")
			fmt.Fprintf(os.Stderr, "  3. API might not have permission to access this collection\n")
			fmt.Fprintf(os.Stderr, "\nTo list all collections, run:\n")
			fmt.Fprintf(os.Stderr, "  go run cmd/list-collections/main.go\n")
			os.Exit(1)
		}

		collection.ID = result.Collection.ID
		collection.Title = result.Collection.Title
		collection.Handle = result.Collection.Handle
		fmt.Printf("✅ Found collection: %s (handle: %s)\n\n", collection.Title, collection.Handle)
	} else {
		// Find collection by title
		fmt.Printf("🔍 Looking for collection: %s\n\n", collectionTitle)

		variables := map[string]interface{}{
			"first": 10,
			"query": fmt.Sprintf(`title:"%s"`, collectionTitle),
		}

		resp, err := client.Execute(CollectionQuery, variables)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query collections: %v\n", err)
			os.Exit(1)
		}

		var collectionResult struct {
			Data struct {
				Collections struct {
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

		if err := json.Unmarshal(resp.Data, &collectionResult); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse collection response: %v\n", err)
			os.Exit(1)
		}

		if len(collectionResult.Data.Collections.Edges) == 0 {
			fmt.Fprintf(os.Stderr, " Collection '%s' not found!\n", collectionTitle)
			fmt.Println("\nAvailable collections might have different titles.")
			fmt.Println("You can also use collection ID: go run cmd/map-partner-catalog/main.go <collection-id>")
			os.Exit(1)
		}

		collectionNode := collectionResult.Data.Collections.Edges[0].Node
		collection.ID = collectionNode.ID
		collection.Title = collectionNode.Title
		collection.Handle = collectionNode.Handle
		fmt.Printf("Found collection: %s (handle: %s)\n\n", collection.Title, collection.Handle)
	}

	// Step 2: Fetch all products from the collection
	fmt.Println("Fetching products from collection...")
	allProducts := []ProductInfo{}
	hasNextPage := true
	after := ""

	for hasNextPage {
		variables := map[string]interface{}{
			"collectionId": collection.ID,
			"first":        50,
		}
		if after != "" {
			variables["after"] = after
		}

		resp, err := client.Execute(ProductsByCollectionQuery, variables)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query products: %v\n", err)
			os.Exit(1)
		}

		var result struct {
			Collection struct {
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
			} `json:"collection"`
		}

		if err := json.Unmarshal(resp.Data, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse products response: %v\n", err)
			os.Exit(1)
		}

		// Extract products with SKUs
		for _, productEdge := range result.Collection.Products.Edges {
			product := productEdge.Node
			productID := extractIDFromGID(product.ID)

			for _, variantEdge := range product.Variants.Edges {
				variant := variantEdge.Node
				if variant.SKU != "" {
					variantID := extractIDFromGID(variant.ID)
					allProducts = append(allProducts, ProductInfo{
						ProductID:   productID,
						VariantID:   variantID,
						SKU:         variant.SKU,
						ProductName: product.Title,
						VariantName: variant.Title,
						Price:       variant.Price,
					})
				}
			}
		}

		hasNextPage = result.Collection.Products.PageInfo.HasNextPage
		after = result.Collection.Products.PageInfo.EndCursor

		if hasNextPage {
			fmt.Printf(" Fetched %d products with SKUs so far...\r", len(allProducts))
		}
	}

	fmt.Printf("\n\n Found %d products with SKUs in collection '%s'\n\n", len(allProducts), collection.Title)

	if len(allProducts) == 0 {
		fmt.Println(" No products with SKUs found in this collection.")
		fmt.Println("   Products need to have SKUs assigned to be mapped.")
		os.Exit(0)
	}

	// Step 3: Connect to database and create mappings
	fmt.Println("Connecting to database...")
	db, err := postgres.NewConnection(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	repos := postgres.NewRepositories(db, logger)
	ctx := context.Background()

	var partnerUUID *uuid.UUID
	if partnerName != "" {
		partners, err := repos.Partner.List(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list partners: %v\n", err)
			os.Exit(1)
		}
		for _, p := range partners {
			if strings.EqualFold(p.Name, partnerName) {
				partnerUUID = &p.ID
				fmt.Printf("Using partner: %s (ID: %s)\n\n", p.Name, p.ID.String())
				break
			}
		}
		if partnerUUID == nil {
			fmt.Fprintf(os.Stderr, "Partner %q not found. List partners: go run cmd/list-partners/main.go\n", partnerName)
			os.Exit(1)
		}
	}

	// Step 4: Create mappings (global sku_mappings or partner_sku_mappings)
	fmt.Printf("Creating SKU mappings...\n\n")
	successCount := 0
	errorCount := 0

	if partnerUUID != nil {
		// Write to partner_sku_mappings for this partner (so cart submit can create orders)
		var mappings []*domain.PartnerSKUMapping
		for _, product := range allProducts {
			title := product.ProductName
			if product.VariantName != "" {
				title = product.VariantName
			}
			m := &domain.PartnerSKUMapping{
				PartnerID:        *partnerUUID,
				SKU:              product.SKU,
				ShopifyProductID: product.ProductID,
				ShopifyVariantID: product.VariantID,
				Title:            &title,
				Price:            &product.Price,
				IsActive:         true,
			}
			mappings = append(mappings, m)
		}
		if err := repos.PartnerSKUMapping.UpsertBatch(ctx, *partnerUUID, mappings); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to upsert partner SKU mappings: %v\n", err)
			os.Exit(1)
		}
		successCount = len(mappings)
	} else {
		for i, product := range allProducts {
			mapping := &domain.SKUMapping{
				SKU:              product.SKU,
				ShopifyProductID: product.ProductID,
				ShopifyVariantID: product.VariantID,
				IsActive:         true,
			}
			err := repos.SKUMapping.Upsert(ctx, mapping)
			if err != nil {
				fmt.Printf(" Failed to map SKU '%s': %v\n", product.SKU, err)
				errorCount++
			} else {
				successCount++
				if (i+1)%10 == 0 || i == len(allProducts)-1 {
					fmt.Printf(" Mapped %d/%d SKUs...\r", successCount, len(allProducts))
				}
			}
		}
	}

	fmt.Printf("\n\n")
	fmt.Println("========================================")
	fmt.Println("  Mapping Summary")
	fmt.Println("========================================")
	fmt.Printf("Successfully mapped: %d SKUs\n", successCount)
	if errorCount > 0 {
		fmt.Printf("Failed to map: %d SKUs\n", errorCount)
	}
	fmt.Println("")
	if partnerUUID != nil {
		fmt.Println("Partner catalog populated. Cart submit with these SKUs will now create orders.")
		fmt.Println("Verify: GET /v1/catalog/products (with partner API key)")
	} else {
		fmt.Println("You can verify mappings with:")
		fmt.Println("   go run cmd/list-sku-mappings/main.go")
	}
	fmt.Println("")
}

func extractIDFromGID(gid string) int64 {
	parts := strings.Split(gid, "/")
	if len(parts) < 4 {
		return 0
	}

	var id int64
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			var err error
			id, err = parseInt64(parts[i])
			if err == nil {
				return id
			}
		}
	}
	return 0
}

func parseInt64(s string) (int64, error) {
	var result int64
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int64(r-'0')
		} else {
			return 0, fmt.Errorf("invalid number")
		}
	}
	return result, nil
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
