package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/shopify"
	"go.uber.org/zap"
)

const ShopInfoQuery = `
query {
  shop {
    name
    myshopifyDomain
  }
}
`

const VariantSearchQuery = `
query productVariantsBySku($first: Int!, $query: String!) {
  productVariants(first: $first, query: $query) {
    edges {
      node {
        id
        sku
        title
        price
        product { id title handle }
      }
    }
  }
}
`

const ProductsTitleSearchQuery = `
query productsByTitle($first: Int!, $query: String!) {
  products(first: $first, query: $query) {
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

type variantNode struct {
	ID      string `json:"id"`
	SKU     string `json:"sku"`
	Title   string `json:"title"`
	Price   string `json:"price"`
	Product struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Handle string `json:"handle"`
	} `json:"product"`
}

var debugMode bool

func main() {
	limit := flag.Int("limit", 25, "How many candidates to request from Shopify search")
	showHex := flag.Bool("hex", false, "Print SKU bytes as hex (useful for hidden characters)")
	debug := flag.Bool("debug", false, "Print debug information (queries and responses)")
	flag.Parse()

	debugMode = *debug

	if flag.NArg() < 1 {
		fmt.Println("Usage: go run cmd/find-sku/main.go [--limit=25] [--hex] [--debug] <sku>")
		os.Exit(1)
	}

	targetSKU := strings.TrimSpace(flag.Arg(0))
	if targetSKU == "" {
		fmt.Fprintln(os.Stderr, "SKU cannot be empty.")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	client := shopify.NewClient(cfg.Shopify, logger)

	// 0) Confirm store identity
	if err := printShopIdentity(client); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch shop identity: %v\n", err)
		fmt.Fprintln(os.Stderr, "This usually indicates wrong endpoint/token/scopes.")
		os.Exit(1)
	}

	fmt.Printf("\nSearching for EXACT SKU (TrimSpace equality): %q\n\n", targetSKU)

	// 1) Phrase query
	phraseQuery := buildPhraseSkuQuery(targetSKU)
	fmt.Printf("1) Phrase query: %q\n", phraseQuery)
	phraseCandidates, err := fetchVariants(client, *limit, phraseQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Shopify phrase query failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   -> %d candidates\n\n", len(phraseCandidates))

	if hit, ok := pickExact(phraseCandidates, targetSKU); ok {
		printHitAndExit(hit, targetSKU)
		return
	}

	// 2) Token query
	tokenQuery := buildTokenSkuQuery(targetSKU)
	fmt.Printf("2) Token query: %q\n", tokenQuery)
	tokenCandidates, err := fetchVariants(client, *limit, tokenQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Shopify token query failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   -> %d candidates\n", len(tokenCandidates))

	if len(tokenCandidates) > 0 {
		fmt.Printf("\nCandidates (none accepted unless EXACT match):\n")
		printCandidates(tokenCandidates, *showHex)
		if hit, ok := pickExact(tokenCandidates, targetSKU); ok {
			fmt.Println()
			printHitAndExit(hit, targetSKU)
			return
		}
		fmt.Printf("\nNOT FOUND (exact): candidates exist, but none had sku exactly %q\n", targetSKU)
		os.Exit(1)
	}

	// 3) If SKU searches both returned 0, prove whether the text exists elsewhere (likely title).
	fmt.Printf("\nSKU searches returned 0. Checking if the text exists in PRODUCT TITLES...\n")
	titleQuery := buildTitleQuery(targetSKU)
	products, err := searchProductsByTitle(client, 5, titleQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Title search failed: %v\n", err)
		os.Exit(1)
	}
	if len(products) == 0 {
		fmt.Printf("No products matched title query %q either.\n", titleQuery)
		fmt.Println("\nConclusion:")
		fmt.Println("  - You are likely querying a different store/environment than where you tested GraphQL, OR")
		fmt.Println("  - The value is not present in Shopify at all.")
		os.Exit(1)
	}

	fmt.Printf("Found %d product(s) matching title query %q:\n", len(products), titleQuery)
	for i, p := range products {
		fmt.Printf("  %d) %s (handle=%s)\n", i+1, p.Title, p.Handle)
	}

	fmt.Println("\nConclusion:")
	fmt.Println("  - The text exists in titles, but NOT in SKU. If you want SKU lookup, you must set SKU on the variant.")
	os.Exit(1)
}

func printShopIdentity(client *shopify.Client) error {
	resp, err := client.Execute(ShopInfoQuery, nil)
	if err != nil {
		return err
	}

	// Debug: print raw response
	if debugMode {
		fmt.Printf("DEBUG: Shop info raw response: %s\n", string(resp.Data))
	}

	// resp.Data is already the "data" object from GraphQL response
	var shopData struct {
		Shop struct {
			Name            string `json:"name"`
			MyshopifyDomain string `json:"myshopifyDomain"`
		} `json:"shop"`
	}
	if err := json.Unmarshal(resp.Data, &shopData); err != nil {
		if debugMode {
			fmt.Printf("DEBUG: Failed to parse shop info: %v\n", err)
		}
		return err
	}
	fmt.Println("Connected Shopify store:")
	fmt.Printf("  Name: %s\n", shopData.Shop.Name)
	fmt.Printf("  Domain: %s\n", shopData.Shop.MyshopifyDomain)
	return nil
}

func fetchVariants(client *shopify.Client, first int, queryStr string) ([]variantNode, error) {
	variables := map[string]any{
		"first": first,
		"query": queryStr,
	}

	// Debug: print what we're sending
	if debugMode {
		fmt.Printf("DEBUG: Sending query with variables: first=%d, query=%q\n", first, queryStr)
	}

	resp, err := client.Execute(VariantSearchQuery, variables)
	if err != nil {
		if debugMode {
			fmt.Printf("DEBUG: Query execution error: %v\n", err)
		}
		return nil, err
	}

	// Debug: print raw response (first 500 chars)
	if debugMode {
		rawResp := string(resp.Data)
		if len(rawResp) > 500 {
			fmt.Printf("DEBUG: Raw response (truncated): %s...\n", rawResp[:500])
		} else {
			fmt.Printf("DEBUG: Raw response: %s\n", rawResp)
		}
	}

	// resp.Data is already the "data" object from GraphQL response
	var parsed struct {
		ProductVariants struct {
			Edges []struct {
				Node variantNode `json:"node"`
			} `json:"edges"`
		} `json:"productVariants"`
	}
	if err := json.Unmarshal(resp.Data, &parsed); err != nil {
		if debugMode {
			fmt.Printf("DEBUG: Parse error: %v\n", err)
			fmt.Printf("DEBUG: Response data: %s\n", string(resp.Data))
		}
		return nil, err
	}

	out := make([]variantNode, 0, len(parsed.ProductVariants.Edges))
	for _, e := range parsed.ProductVariants.Edges {
		out = append(out, e.Node)
	}
	return out, nil
}

func pickExact(cands []variantNode, targetSKU string) (variantNode, bool) {
	for _, v := range cands {
		if strings.TrimSpace(v.SKU) == targetSKU {
			return v, true
		}
	}
	return variantNode{}, false
}

func printHitAndExit(v variantNode, targetSKU string) {
	productID := extractIDFromGID(v.Product.ID)
	variantID := extractIDFromGID(v.ID)

	fmt.Println("\nFOUND (exact match):")
	fmt.Printf("  SKU           : %q\n", strings.TrimSpace(v.SKU))
	fmt.Printf("  Product Title : %s\n", v.Product.Title)
	fmt.Printf("  Handle        : %s\n", v.Product.Handle)
	fmt.Printf("  Variant Title : %s\n", v.Title)
	fmt.Printf("  Price         : %s\n", v.Price)

	fmt.Println("\nIDs:")
	fmt.Printf("  Product ID: %d\n", productID)
	fmt.Printf("  Variant ID: %d\n", variantID)

	fmt.Println("\nTo add this to the database, run:")
	fmt.Printf("  go run cmd/add-sku/main.go %q %d %d\n", targetSKU, productID, variantID)
}

func printCandidates(cands []variantNode, showHex bool) {
	seen := make(map[string]struct{})
	i := 0
	for _, v := range cands {
		key := v.Product.ID + "|" + v.ID + "|" + v.SKU
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		i++

		raw := strings.TrimSpace(v.SKU)
		visible := makeVisible(raw)

		fmt.Printf("  %d) sku=%q | visible=%q | product=%q | variant=%q\n",
			i, raw, visible, v.Product.Title, v.Title)

		if showHex {
			fmt.Printf("     bytes(hex)=%s\n", hex.EncodeToString([]byte(raw)))
		}
	}
}

func makeVisible(s string) string {
	s = strings.ReplaceAll(s, " ", "·")
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

func buildPhraseSkuQuery(sku string) string {
	s := strings.ReplaceAll(sku, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `sku:"` + s + `"`
}

func buildTokenSkuQuery(sku string) string {
	s := strings.ReplaceAll(sku, `"`, `\"`)
	return `sku:` + s
}

type productHit struct {
	Title  string `json:"title"`
	Handle string `json:"handle"`
}

func buildTitleQuery(q string) string {
	// Shopify supports product search via query string; title:* is common, but simple text works too.
	// We quote to tighten it.
	s := strings.ReplaceAll(q, `"`, `\"`)
	return `title:"` + s + `"`
}

func searchProductsByTitle(client *shopify.Client, first int, queryStr string) ([]productHit, error) {
	variables := map[string]any{
		"first": first,
		"query": queryStr,
	}
	resp, err := client.Execute(ProductsTitleSearchQuery, variables)
	if err != nil {
		return nil, err
	}
	// resp.Data is already the "data" object from GraphQL response
	var parsed struct {
		Products struct {
			Edges []struct {
				Node productHit `json:"node"`
			} `json:"edges"`
		} `json:"products"`
	}
	if err := json.Unmarshal(resp.Data, &parsed); err != nil {
		return nil, err
	}
	out := make([]productHit, 0, len(parsed.Products.Edges))
	for _, e := range parsed.Products.Edges {
		out = append(out, e.Node)
	}
	return out, nil
}

func extractIDFromGID(gid string) int64 {
	start := -1
	end := -1
	for i := len(gid) - 1; i >= 0; i-- {
		c := gid[i]
		if c >= '0' && c <= '9' {
			if end == -1 {
				end = i
			}
			start = i
		} else if end != -1 {
			break
		}
	}
	if start == -1 || end == -1 {
		return 0
	}
	var id int64
	for i := start; i <= end; i++ {
		id = id*10 + int64(gid[i]-'0')
	}
	return id
}
