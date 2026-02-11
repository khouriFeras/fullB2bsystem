package productb2b

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client calls ProductB2B API with service key
type Client struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a ProductB2B HTTP client
func NewClient(baseURL, serviceKey string, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		serviceKey: serviceKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// GetCatalogProducts fetches products for a collection (paginated)
// Returns raw response body and error. If ProductB2B is down, returns (nil, error).
func (c *Client) GetCatalogProducts(ctx context.Context, collectionHandle, cursor string, limit int) ([]byte, error) {
	if c.baseURL == "" || c.serviceKey == "" {
		return nil, fmt.Errorf("productb2b client not configured: base URL and service key required")
	}
	u, err := url.Parse(c.baseURL + "/v1/catalog/products")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("collection_handle", collectionHandle)
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("ProductB2B catalog request failed", zap.Error(err), zap.String("collection_handle", collectionHandle))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("productb2b returned %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// GetProductBySKU fetches a single product by SKU (for order enrichment)
func (c *Client) GetProductBySKU(ctx context.Context, sku string) ([]byte, error) {
	if c.baseURL == "" || c.serviceKey == "" {
		return nil, fmt.Errorf("productb2b client not configured")
	}
	u, err := url.Parse(c.baseURL + "/v1/catalog/products")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("sku", sku)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("ProductB2B product by SKU request failed", zap.Error(err), zap.String("sku", sku))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("productb2b returned %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// CatalogProductsResponse is the ProductB2B catalog response shape (subset for parsing)
type CatalogProductsResponse struct {
	Data       []map[string]interface{} `json:"data"`
	Pagination map[string]interface{}   `json:"pagination"`
	Meta       map[string]interface{}   `json:"meta"`
}

// ParseCatalogProducts parses raw JSON into data + pagination
func ParseCatalogProducts(raw []byte) (*CatalogProductsResponse, error) {
	var out CatalogProductsResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// parseGIDToInt64 extracts numeric ID from Shopify GID (e.g. gid://shopify/Product/123 -> 123)
func parseGIDToInt64(gid string) int64 {
	parts := strings.Split(gid, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "" {
			continue
		}
		var n int64
		if _, err := fmt.Sscanf(parts[i], "%d", &n); err == nil {
			return n
		}
	}
	return 0
}

// ProductVariantInfo is one row for partner_sku_mappings (one per variant with SKU)
type ProductVariantInfo struct {
	SKU              string
	Title            string
	Price            string
	ImageURL         *string
	ShopifyProductID int64
	ShopifyVariantID int64
}

// ExtractProductVariantInfos extracts all variant rows from a ProductB2B product node
func ExtractProductVariantInfos(p map[string]interface{}) (productTitle string, rows []ProductVariantInfo) {
	if t, ok := p["title"].(string); ok {
		productTitle = t
	}
	productID := parseGIDToInt64(getStr(p, "id"))
	var imageURL *string
	if fi, ok := p["featuredImage"].(map[string]interface{}); ok {
		if u, ok := fi["url"].(string); ok {
			imageURL = &u
		}
	}
	// ProductB2B can return variants as [] or { nodes: [] }
	var variants []interface{}
	if vv, ok := p["variants"].([]interface{}); ok {
		variants = vv
	} else if vmap, ok := p["variants"].(map[string]interface{}); ok {
		variants, _ = vmap["nodes"].([]interface{})
	}
	for _, v := range variants {
		vm, _ := v.(map[string]interface{})
		sku := getStr(vm, "sku")
		if sku == "" {
			continue
		}
		price := getStr(vm, "price")
		variantID := parseGIDToInt64(getStr(vm, "id"))
		if variantID == 0 {
			continue
		}
		rows = append(rows, ProductVariantInfo{
			SKU:              sku,
			Title:            productTitle,
			Price:            price,
			ImageURL:         imageURL,
			ShopifyProductID: productID,
			ShopifyVariantID: variantID,
		})
	}
	return productTitle, rows
}

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
