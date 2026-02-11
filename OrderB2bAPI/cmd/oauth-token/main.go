package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run cmd/oauth-token/main.go <shop-domain> <client-id> <client-secret>")
		fmt.Println("Example: go run cmd/oauth-token/main.go jafarshop.myshopify.com shpca_xxxxx shpss_xxxxx")
		fmt.Println("\nNote: This requires manual authorization. Follow the steps:")
		fmt.Println("1. Run this script - it will give you an authorization URL")
		fmt.Println("2. Visit the URL in your browser and authorize")
		fmt.Println("3. Copy the 'code' from the redirect URL")
		fmt.Println("4. Run the script again with the code")
		os.Exit(1)
	}

	shopDomain := os.Args[1]
	clientID := os.Args[2]
	clientSecret := os.Args[3]

	// Remove https:// if present
	shopDomain = strings.TrimPrefix(shopDomain, "https://")
	shopDomain = strings.TrimPrefix(shopDomain, "http://")

	// Check if we have a code (step 2)
	if len(os.Args) >= 5 {
		code := os.Args[4]
		accessToken, err := exchangeCodeForToken(shopDomain, clientID, clientSecret, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get access token: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Access Token obtained!\n\n")
		fmt.Printf("Add this to your .env file:\n")
		fmt.Printf("SHOPIFY_ACCESS_TOKEN=%s\n", accessToken)
		return
	}

	// Step 1: Generate authorization URL
	authURL := fmt.Sprintf("https://%s/admin/oauth/authorize?client_id=%s&scope=read_products,write_draft_orders&redirect_uri=urn:ietf:wg:oauth:2.0:oob",
		shopDomain, clientID)

	fmt.Printf("Step 1: Authorize the app\n\n")
	fmt.Printf("Visit this URL in your browser:\n")
	fmt.Printf("%s\n\n", authURL)
	fmt.Printf("After authorizing, you'll get a code.\n")
	fmt.Printf("Then run:\n")
	fmt.Printf("go run cmd/oauth-token/main.go %s %s %s <code>\n", shopDomain, clientID, clientSecret)
}

func exchangeCodeForToken(shopDomain, clientID, clientSecret, code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/admin/oauth/access_token", shopDomain),
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}
