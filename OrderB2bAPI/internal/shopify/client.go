package shopify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/config"
)

type Client struct {
	shopDomain  string
	accessToken string
	apiVersion  string
	httpClient  *http.Client
	logger      *zap.Logger
}

// NewClient creates a new Shopify GraphQL client
func NewClient(cfg config.ShopifyConfig, logger *zap.Logger) *Client {
	// Normalize shop domain - remove https://, http://, and trailing slashes
	shopDomain := cfg.ShopDomain
	shopDomain = strings.TrimPrefix(shopDomain, "https://")
	shopDomain = strings.TrimPrefix(shopDomain, "http://")
	shopDomain = strings.TrimSuffix(shopDomain, "/")
	
	return &Client{
		shopDomain:  shopDomain,
		accessToken: cfg.AccessToken,
		apiVersion:  cfg.APIVersion,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   json.RawMessage        `json:"data"`
	Errors []GraphQLError         `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string        `json:"message"`
	Path    []interface{} `json:"path,omitempty"`
}

// Execute executes a GraphQL query/mutation
func (c *Client) Execute(query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	url := fmt.Sprintf("https://%s/admin/api/%s/graphql.json", c.shopDomain, c.apiVersion)

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Shopify-Access-Token", c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("shopify API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var graphQLResp GraphQLResponse
	if err := json.Unmarshal(body, &graphQLResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(body))
	}

	if len(graphQLResp.Errors) > 0 {
		errorMessages := make([]string, len(graphQLResp.Errors))
		for i, err := range graphQLResp.Errors {
			errorMessages[i] = err.Message
		}
		return nil, fmt.Errorf("graphQL errors: %s", strings.Join(errorMessages, "; "))
	}

	return &graphQLResp, nil
}
