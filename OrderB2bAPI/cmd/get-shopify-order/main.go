package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/get-shopify-order/main.go <order_number>")
		fmt.Println("Example: go run cmd/get-shopify-order/main.go \"#45246\"")
		fmt.Println("Example: go run cmd/get-shopify-order/main.go \"name:#45246\"")
		os.Exit(1)
	}

	orderNumber := os.Args[1]
	
	// If order number starts with #, format as Shopify query
	queryString := orderNumber
	if len(orderNumber) > 0 && orderNumber[0] == '#' {
		queryString = fmt.Sprintf("name:%s", orderNumber)
	} else if !contains(orderNumber, ":") {
		queryString = fmt.Sprintf("name:#%s", orderNumber)
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

	fmt.Printf("🔍 Fetching order from Shopify: %s\n\n", orderNumber)

	// Build query string dynamically (GraphQL doesn't allow variables in query parameter)
	query := fmt.Sprintf(shopify.OrderByNumberQueryTemplate, queryString)

	// Execute query (no variables needed)
	resp, err := client.Execute(query, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to query Shopify: %v\n", err)
		os.Exit(1)
	}

	// Parse response
	var result struct {
		Data struct {
			Orders struct {
				Edges []struct {
					Node struct {
						ID                        string `json:"id"`
						Name                      string `json:"name"`
						DisplayFulfillmentStatus  string `json:"displayFulfillmentStatus"`
						DisplayFinancialStatus    string `json:"displayFinancialStatus"`
						CreatedAt                 string `json:"createdAt"`
						UpdatedAt                 string `json:"updatedAt"`
						TotalPriceSet             struct {
							ShopMoney struct {
								Amount       string `json:"amount"`
								CurrencyCode string `json:"currencyCode"`
							} `json:"shopMoney"`
						} `json:"totalPriceSet"`
						Customer struct {
							FirstName string `json:"firstName"`
							LastName  string `json:"lastName"`
							Email     string `json:"email"`
							Phone     string `json:"phone"`
						} `json:"customer"`
						ShippingAddress struct {
							Address1 string `json:"address1"`
							Address2 string `json:"address2"`
							City     string `json:"city"`
							Province string `json:"province"`
							Zip      string `json:"zip"`
							Country  string `json:"country"`
						} `json:"shippingAddress"`
						LineItems struct {
							Edges []struct {
								Node struct {
									ID       string `json:"id"`
									Title    string `json:"title"`
									Quantity int    `json:"quantity"`
									Variant  *struct {
										ID    string `json:"id"`
										SKU   string `json:"sku"`
										Title string `json:"title"`
										Price string `json:"price"`
									} `json:"variant"`
									OriginalUnitPriceSet struct {
										ShopMoney struct {
											Amount       string `json:"amount"`
											CurrencyCode string `json:"currencyCode"`
										} `json:"shopMoney"`
									} `json:"originalUnitPriceSet"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"lineItems"`
						Fulfillments []struct {
							ID         string `json:"id"`
							Status     string `json:"status"`
							TrackingInfo []struct {
								Number  string `json:"number"`
								URL     string `json:"url"`
								Company string `json:"company"`
							} `json:"trackingInfo"`
						} `json:"fulfillments"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"orders"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to parse response: %v\n", err)
		fmt.Fprintf(os.Stderr, "Response: %s\n", string(resp.Data))
		os.Exit(1)
	}

	if len(result.Data.Orders.Edges) == 0 {
		fmt.Printf("❌ Order not found in Shopify\n")
		os.Exit(1)
	}

	order := result.Data.Orders.Edges[0].Node

	fmt.Printf("✅ Order found!\n\n")
	fmt.Printf("Order Information:\n")
	fmt.Printf("  Order Number: %s\n", order.Name)
	fmt.Printf("  Order ID: %s\n", order.ID)
	fmt.Printf("  Fulfillment Status: %s\n", order.DisplayFulfillmentStatus)
	fmt.Printf("  Financial Status: %s\n", order.DisplayFinancialStatus)
	fmt.Printf("  Total: %s %s\n", order.TotalPriceSet.ShopMoney.Amount, order.TotalPriceSet.ShopMoney.CurrencyCode)
	fmt.Printf("  Created: %s\n", order.CreatedAt)
	fmt.Printf("  Updated: %s\n", order.UpdatedAt)
	
	if order.Customer.FirstName != "" || order.Customer.LastName != "" {
		fmt.Printf("\nCustomer:\n")
		fmt.Printf("  Name: %s %s\n", order.Customer.FirstName, order.Customer.LastName)
		if order.Customer.Email != "" {
			fmt.Printf("  Email: %s\n", order.Customer.Email)
		}
		if order.Customer.Phone != "" {
			fmt.Printf("  Phone: %s\n", order.Customer.Phone)
		}
	}

	if order.ShippingAddress.Address1 != "" {
		fmt.Printf("\nShipping Address:\n")
		fmt.Printf("  %s\n", order.ShippingAddress.Address1)
		if order.ShippingAddress.Address2 != "" {
			fmt.Printf("  %s\n", order.ShippingAddress.Address2)
		}
		fmt.Printf("  %s, %s %s\n", order.ShippingAddress.City, order.ShippingAddress.Province, order.ShippingAddress.Zip)
		fmt.Printf("  %s\n", order.ShippingAddress.Country)
	}

	if len(order.LineItems.Edges) > 0 {
		fmt.Printf("\nLine Items:\n")
		for i, item := range order.LineItems.Edges {
			fmt.Printf("  %d. %s (x%d)\n", i+1, item.Node.Title, item.Node.Quantity)
			if item.Node.Variant != nil {
				if item.Node.Variant.SKU != "" {
					fmt.Printf("     SKU: %s\n", item.Node.Variant.SKU)
				}
			}
			fmt.Printf("     Price: %s %s\n", item.Node.OriginalUnitPriceSet.ShopMoney.Amount, item.Node.OriginalUnitPriceSet.ShopMoney.CurrencyCode)
		}
	}

	if len(order.Fulfillments) > 0 {
		fmt.Printf("\nFulfillments:\n")
		for i, fulfillment := range order.Fulfillments {
			fmt.Printf("  %d. Status: %s\n", i+1, fulfillment.Status)
			for _, tracking := range fulfillment.TrackingInfo {
				if tracking.Number != "" {
					fmt.Printf("     Tracking: %s (%s)\n", tracking.Number, tracking.Company)
				}
				if tracking.URL != "" {
					fmt.Printf("     URL: %s\n", tracking.URL)
				}
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
