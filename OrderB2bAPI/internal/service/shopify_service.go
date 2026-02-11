package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/shopify"
)

type shopifyService struct {
	client *shopify.Client
	repos  *repository.Repositories
	logger *zap.Logger
}

// NewShopifyService creates a new Shopify service
func NewShopifyService(cfg config.ShopifyConfig, repos *repository.Repositories, logger *zap.Logger) *shopifyService {
	return &shopifyService{
		client: shopify.NewClient(cfg, logger),
		repos:  repos,
		logger: logger,
	}
}

// CompleteDraftOrder completes a Shopify draft order and returns the Shopify Order numeric ID.
func (s *shopifyService) CompleteDraftOrder(ctx context.Context, draftOrderID int64) (int64, error) {
	draftOrderGID := fmt.Sprintf("gid://shopify/DraftOrder/%d", draftOrderID)
	variables := map[string]interface{}{
		"id": draftOrderGID,
	}

	resp, err := s.client.Execute(shopify.DraftOrderCompleteMutation, variables)
	if err != nil {
		return 0, fmt.Errorf("failed to complete draft order: %w", err)
	}

	// resp.Data is already the "data" object from GraphQL response
	var result struct {
		DraftOrderComplete struct {
			DraftOrder struct {
				ID    string `json:"id"`
				Order struct {
					ID string `json:"id"`
				} `json:"order"`
			} `json:"draftOrder"`
			UserErrors []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"draftOrderComplete"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return 0, fmt.Errorf("failed to parse draft order complete response: %w", err)
	}

	if len(result.DraftOrderComplete.UserErrors) > 0 {
		return 0, fmt.Errorf("shopify user errors: %v", result.DraftOrderComplete.UserErrors)
	}

	// Extract numeric Order ID from GID (gid://shopify/Order/123)
	orderGID := result.DraftOrderComplete.DraftOrder.Order.ID
	orderID, err := extractIDFromGID(orderGID)
	if err != nil {
		return 0, fmt.Errorf("failed to extract order ID: %w", err)
	}
	return orderID, nil
}

// GetOrderNameByID fetches the order by numeric ID and returns its display name (e.g. "#1033").
func (s *shopifyService) GetOrderNameByID(ctx context.Context, orderID int64) (string, error) {
	orderGID := fmt.Sprintf("gid://shopify/Order/%d", orderID)
	variables := map[string]interface{}{"id": orderGID}
	resp, err := s.client.Execute(shopify.OrderByIDQuery, variables)
	if err != nil {
		return "", fmt.Errorf("get order by ID: %w", err)
	}
	var result struct {
		Node struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"node"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return "", fmt.Errorf("parse order response: %w", err)
	}
	if result.Node.Name == "" {
		return "", fmt.Errorf("order has no name")
	}
	// Store without # (e.g. "1033" not "#1033")
	name := strings.TrimPrefix(result.Node.Name, "#")
	return name, nil
}

// GetOrderGIDByName looks up an order by its number (e.g. "1033", stored without #) and returns its GID.
func (s *shopifyService) GetOrderGIDByName(ctx context.Context, orderName string) (string, error) {
	// Shopify query expects "name:#1033"
	queryName := orderName
	if queryName != "" && !strings.HasPrefix(queryName, "#") {
		queryName = "#" + queryName
	}
	queryStr := fmt.Sprintf(shopify.OrderByNumberQueryTemplate, "name:"+queryName)
	resp, err := s.client.Execute(queryStr, nil)
	if err != nil {
		return "", fmt.Errorf("get order by number: %w", err)
	}
	var result struct {
		Orders struct {
			Edges []struct {
				Node struct {
					ID string `json:"id"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return "", fmt.Errorf("parse orders response: %w", err)
	}
	if len(result.Orders.Edges) == 0 {
		return "", fmt.Errorf("order not found: %s", orderName)
	}
	return result.Orders.Edges[0].Node.ID, nil
}

// SetOrderPartnerMetafield sets the custom.parnters metafield on a Shopify Order (by order name, e.g. "#1033").
func (s *shopifyService) SetOrderPartnerMetafield(ctx context.Context, shopifyOrderName string, partnerName string) error {
	orderGID, err := s.GetOrderGIDByName(ctx, shopifyOrderName)
	if err != nil {
		return fmt.Errorf("resolve order by name: %w", err)
	}
	metafields := []shopify.MetafieldsSetInput{
		{
			OwnerID:   orderGID,
			Namespace: "custom",
			Key:       "parnters",
			Type:      "single_line_text_field",
			Value:     partnerName,
		},
	}
	variables := map[string]interface{}{
		"metafields": metafields,
	}
	resp, err := s.client.Execute(shopify.MetafieldsSetMutation, variables)
	if err != nil {
		return fmt.Errorf("metafieldsSet: %w", err)
	}
	var result struct {
		MetafieldsSet struct {
			UserErrors []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
				Code    string   `json:"code"`
			} `json:"userErrors"`
		} `json:"metafieldsSet"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return fmt.Errorf("parse metafieldsSet response: %w", err)
	}
	if len(result.MetafieldsSet.UserErrors) > 0 {
		return fmt.Errorf("metafieldsSet userErrors: %v", result.MetafieldsSet.UserErrors)
	}
	s.logger.Info("Set order metafield custom.parnters", zap.String("shopify_order_name", shopifyOrderName), zap.String("partner", partnerName))
	return nil
}

// FindCustomerIDByPartnerOrderTag looks up Shopify for an order with the same partner order tag.
// If found, returns that order's customer GID and phone so the new draft can be attached to the same customer
// only when the request phone matches (same order number + same phone = same customer).
// Returns nil,nil if no matching order or caller should use email to create/find customer.
func (s *shopifyService) FindCustomerIDByPartnerOrderTag(ctx context.Context, partnerOrderID string) (customerID *string, customerPhone *string, err error) {
	if partnerOrderID == "" {
		return nil, nil, nil
	}
	// Search by tag we set on draft orders: "partner_order:<id>"
	queryString := fmt.Sprintf("tag:partner_order:%s", strings.ReplaceAll(partnerOrderID, " ", "_"))
	query := fmt.Sprintf(shopify.OrderByNumberQueryTemplate, queryString)
	resp, err := s.client.Execute(query, nil)
	if err != nil {
		s.logger.Debug("FindCustomerIDByPartnerOrderTag: Shopify query failed", zap.String("partner_order_id", partnerOrderID), zap.Error(err))
		return nil, nil, err
	}
	var result struct {
		Orders struct {
			Edges []struct {
				Node struct {
					Customer *struct {
						ID    string `json:"id"`
						Phone string `json:"phone"`
					} `json:"customer"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, nil, fmt.Errorf("parse order lookup response: %w", err)
	}
	if len(result.Orders.Edges) == 0 {
		return nil, nil, nil
	}
	node := result.Orders.Edges[0].Node
	if node.Customer == nil || node.Customer.ID == "" {
		return nil, nil, nil
	}
	id := node.Customer.ID
	phone := strings.TrimSpace(node.Customer.Phone)
	if phone != "" {
		customerPhone = &phone
	}
	customerID = &id
	s.logger.Info("Found existing Shopify order for partner order", zap.String("partner_order_id", partnerOrderID), zap.String("customer_id", id), zap.String("customer_phone", phone))
	return customerID, customerPhone, nil
}

// GetOrderNameByPartnerOrderTag looks up Shopify for an order with the partner_order tag and returns its order name (e.g. "1034" without #).
// Used as fallback for delivery-status when shopify_order_id is not stored so Wassel can be queried by Shopify order number.
func (s *shopifyService) GetOrderNameByPartnerOrderTag(ctx context.Context, partnerOrderID string) (string, error) {
	if partnerOrderID == "" {
		return "", nil
	}
	queryString := fmt.Sprintf("tag:partner_order:%s", strings.ReplaceAll(partnerOrderID, " ", "_"))
	query := fmt.Sprintf(shopify.OrderByNumberQueryTemplate, queryString)
	resp, err := s.client.Execute(query, nil)
	if err != nil {
		return "", fmt.Errorf("shopify order lookup: %w", err)
	}
	var result struct {
		Orders struct {
			Edges []struct {
				Node struct {
					Name string `json:"name"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return "", fmt.Errorf("parse order lookup: %w", err)
	}
	if len(result.Orders.Edges) == 0 {
		return "", nil
	}
	name := strings.TrimSpace(result.Orders.Edges[0].Node.Name)
	if name == "" {
		return "", nil
	}
	return strings.TrimPrefix(name, "#"), nil
}

// FindCustomerIDByPhone looks up a Shopify customer by phone number. Returns the customer GID if found.
// Uses customers query with phone filter (e.g. phone:0778888888 or phone:778888888).
func (s *shopifyService) FindCustomerIDByPhone(ctx context.Context, phone string) (*string, error) {
	if phone == "" {
		return nil, nil
	}
	norm := normalizePhoneForComparison(phone)
	if norm == "" {
		return nil, nil
	}
	// Shopify phone search: try with digits only (e.g. 778888888 or 0778888888)
	queryString := "phone:" + norm
	if len(norm) < 10 {
		queryString = "phone:0" + norm
	}
	query := fmt.Sprintf(shopify.CustomersByPhoneQueryTemplate, queryString)
	resp, err := s.client.Execute(query, nil)
	if err != nil {
		s.logger.Debug("FindCustomerIDByPhone: Shopify query failed", zap.String("phone", phone), zap.Error(err))
		return nil, err
	}
	var result struct {
		Customers struct {
			Edges []struct {
				Node struct {
					ID string `json:"id"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"customers"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("parse customers by phone response: %w", err)
	}
	if len(result.Customers.Edges) == 0 {
		return nil, nil
	}
	id := result.Customers.Edges[0].Node.ID
	s.logger.Info("Found Shopify customer by phone", zap.String("phone", phone), zap.String("customer_id", id))
	return &id, nil
}

// EnsureCustomerAddress fetches the customer's default address in Shopify and, if it differs from the order's shipping address, updates it (or creates one if missing).
func (s *shopifyService) EnsureCustomerAddress(ctx context.Context, customerID string, order *domain.SupplierOrder) error {
	variables := map[string]interface{}{"id": customerID}
	resp, err := s.client.Execute(shopify.CustomerWithAddressesQuery, variables)
	if err != nil {
		return fmt.Errorf("fetch customer addresses: %w", err)
	}
	var result struct {
		Customer *struct {
			ID             string `json:"id"`
			DefaultAddress *struct {
				ID          string `json:"id"`
				Address1    string `json:"address1"`
				City        string `json:"city"`
				Province    string `json:"province"`
				Zip         string `json:"zip"`
				CountryCode string `json:"countryCode"`
			} `json:"defaultAddress"`
			AddressesV2 struct {
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Address1    string `json:"address1"`
						City        string `json:"city"`
						Province    string `json:"province"`
						Zip         string `json:"zip"`
						CountryCode string `json:"countryCode"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"addressesV2"`
		} `json:"customer"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return fmt.Errorf("parse customer addresses response: %w", err)
	}
	if result.Customer == nil {
		return fmt.Errorf("customer not found")
	}

	orderAddr := buildOrderMailingAddress(order)
	def := result.Customer.DefaultAddress
	hasDefault := def != nil && def.ID != ""

	if !hasDefault || len(result.Customer.AddressesV2.Edges) == 0 {
		// No address in Shopify: create one and set as default
		variables := map[string]interface{}{
			"customerId":   customerID,
			"address":      orderAddr,
			"setAsDefault": true,
		}
		resp, err := s.client.Execute(shopify.CustomerAddressCreateMutation, variables)
		if err != nil {
			return fmt.Errorf("customerAddressCreate: %w", err)
		}
		var createResult struct {
			CustomerAddressCreate struct {
				UserErrors []struct {
					Field   []string `json:"field"`
					Message string   `json:"message"`
				} `json:"userErrors"`
			} `json:"customerAddressCreate"`
		}
		if err := json.Unmarshal(resp.Data, &createResult); err != nil {
			return fmt.Errorf("parse customerAddressCreate response: %w", err)
		}
		if len(createResult.CustomerAddressCreate.UserErrors) > 0 {
			return fmt.Errorf("customerAddressCreate userErrors: %v", createResult.CustomerAddressCreate.UserErrors)
		}
		s.logger.Info("Created and set default address for existing customer", zap.String("customer_id", customerID))
		return nil
	}

	// Compare default address with order address (normalized)
	if addressesEqual(def.Address1, def.City, def.Province, def.Zip, def.CountryCode, orderAddr) {
		return nil
	}

	// Update default address to match order
	variables = map[string]interface{}{
		"customerId":   customerID,
		"addressId":    def.ID,
		"address":      orderAddr,
		"setAsDefault": true,
	}
	resp, err = s.client.Execute(shopify.CustomerAddressUpdateMutation, variables)
	if err != nil {
		return fmt.Errorf("customerAddressUpdate: %w", err)
	}
	var updateResult struct {
		CustomerAddressUpdate struct {
			UserErrors []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"customerAddressUpdate"`
	}
	if err := json.Unmarshal(resp.Data, &updateResult); err != nil {
		return fmt.Errorf("parse customerAddressUpdate response: %w", err)
	}
	if len(updateResult.CustomerAddressUpdate.UserErrors) > 0 {
		return fmt.Errorf("customerAddressUpdate userErrors: %v", updateResult.CustomerAddressUpdate.UserErrors)
	}
	s.logger.Info("Updated customer default address to match order", zap.String("customer_id", customerID), zap.String("address_id", def.ID))
	return nil
}

// buildOrderMailingAddress builds MailingAddressInput from order for customer address create/update (countryCode = ISO 3166-1 alpha-2).
func buildOrderMailingAddress(order *domain.SupplierOrder) shopify.MailingAddressInput {
	street := getStringFromMap(order.ShippingAddress, "street")
	city := getStringFromMap(order.ShippingAddress, "city")
	zip := getStringFromMap(order.ShippingAddress, "postal_code")
	country := getStringFromMap(order.ShippingAddress, "country")
	countryCode := countryToCountryCode(country)
	var province *string
	if state, ok := order.ShippingAddress["state"].(string); ok && state != "" {
		province = &state
	}
	nameParts := strings.Fields(order.CustomerName)
	firstName := ""
	lastName := ""
	if len(nameParts) > 0 {
		firstName = nameParts[0]
		if len(nameParts) > 1 {
			lastName = strings.Join(nameParts[1:], " ")
		}
	}
	out := shopify.MailingAddressInput{
		Address1:     street,
		City:         city,
		Zip:          zip,
		CountryCode:  countryCode,
		FirstName:    firstName,
		LastName:     lastName,
		ProvinceCode: province,
	}
	if order.CustomerPhone != "" {
		out.Phone = &order.CustomerPhone
	}
	return out
}

// addressesEqual compares Shopify default address fields with order MailingAddressInput (normalized: trim, lowercase).
func addressesEqual(addr1, city, province, zip, countryCode string, order shopify.MailingAddressInput) bool {
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	op := func(p *string) string {
		if p == nil {
			return ""
		}
		return norm(*p)
	}
	return norm(addr1) == norm(order.Address1) &&
		norm(city) == norm(order.City) &&
		norm(province) == op(order.ProvinceCode) &&
		norm(zip) == norm(order.Zip) &&
		norm(countryCode) == norm(order.CountryCode)
}

// countryToCountryCode maps country name or 2-letter code to ISO 3166-1 alpha-2 (Shopify CountryCode). We only operate in Jordan, so default is JO.
func countryToCountryCode(country string) string {
	c := strings.TrimSpace(country)
	if c == "" {
		return "JO"
	}
	if len(c) == 2 {
		return strings.ToUpper(c)
	}
	if strings.EqualFold(c, "jordan") || strings.EqualFold(c, "jo") {
		return "JO"
	}
	return "JO"
}

// CreateDraftOrder creates a Shopify draft order from a supplier order
func (s *shopifyService) CreateDraftOrder(
	ctx context.Context,
	order *domain.SupplierOrder,
	items []*domain.SupplierOrderItem,
	partnerName string,
) (int64, error) {
	// Build line items
	lineItems := make([]shopify.DraftOrderLineItemInput, 0, len(items))

	for _, item := range items {
		if item.IsSupplierItem && item.ShopifyVariantID != nil {
			// Supplier item - use variant
			variantIDStr := fmt.Sprintf("gid://shopify/ProductVariant/%d", *item.ShopifyVariantID)
			lineItems = append(lineItems, shopify.DraftOrderLineItemInput{
				VariantID: &variantIDStr,
				Quantity:  item.Quantity,
			})
		} else {
			// Non-supplier item - use custom line item
			priceStr := fmt.Sprintf("%.2f", item.Price)
			title := item.Title
			if item.ProductURL != nil {
				title = fmt.Sprintf("%s (URL: %s)", title, *item.ProductURL)
			}

			customAttrs := []shopify.DraftOrderAttributeInput{
				{Key: "product_url", Value: *item.ProductURL},
			}
			if item.ProductURL == nil {
				customAttrs = []shopify.DraftOrderAttributeInput{}
			}

			lineItems = append(lineItems, shopify.DraftOrderLineItemInput{
				Title:             &title,
				OriginalUnitPrice: &priceStr,
				Quantity:          item.Quantity,
				CustomAttributes:  customAttrs,
			})
		}
	}

	// Build shipping address
	shippingAddr := shopify.DraftOrderAddressInput{
		Address1: getStringFromMap(order.ShippingAddress, "street"),
		City:     getStringFromMap(order.ShippingAddress, "city"),
		Zip:      getStringFromMap(order.ShippingAddress, "postal_code"),
		Country:  getStringFromMap(order.ShippingAddress, "country"),
	}

	// Parse customer name (assume "FirstName LastName" or just "Name")
	nameParts := strings.Fields(order.CustomerName)
	if len(nameParts) > 0 {
		shippingAddr.FirstName = nameParts[0]
		if len(nameParts) > 1 {
			lastName := strings.Join(nameParts[1:], " ")
			shippingAddr.LastName = &lastName
		}
	}

	if state, ok := order.ShippingAddress["state"].(string); ok && state != "" {
		shippingAddr.Province = &state
	}

	if order.CustomerPhone != "" {
		shippingAddr.Phone = &order.CustomerPhone
	}

	// Build tags
	tags := []string{
		fmt.Sprintf("partner:%s", partnerName),
		fmt.Sprintf("partner_order:%s", order.PartnerOrderID),
		"pending_confirmation",
	}

	// Check if mixed cart (has both supplier and non-supplier items)
	hasSupplierItems := false
	hasNonSupplierItems := false
	for _, item := range items {
		if item.IsSupplierItem {
			hasSupplierItems = true
		} else {
			hasNonSupplierItems = true
		}
	}

	if hasSupplierItems && hasNonSupplierItems {
		tags = append(tags, "mixed_cart")
	}

	// Customer is determined by phone only: (1) same partner order + same phone, (2) find by phone in Shopify, (3) create with generated email from phone (never use request email for matching).
	existingCustomerID, existingCustomerPhone, err := s.FindCustomerIDByPartnerOrderTag(ctx, order.PartnerOrderID)
	if err != nil {
		s.logger.Warn("Lookup existing customer by partner order tag failed", zap.String("partner_order_id", order.PartnerOrderID), zap.Error(err))
	}

	var useExistingCustomer bool
	if existingCustomerID != nil && existingCustomerPhone != nil && order.CustomerPhone != "" {
		reqNorm := normalizePhoneForComparison(order.CustomerPhone)
		existingNorm := normalizePhoneForComparison(*existingCustomerPhone)
		useExistingCustomer = (reqNorm != "" && existingNorm != "" && reqNorm == existingNorm)
	}

	var customerIDToUse *string
	if useExistingCustomer {
		customerIDToUse = existingCustomerID
		s.logger.Info("Attaching draft order to existing Shopify customer (same order number + same phone)", zap.String("partner_order_id", order.PartnerOrderID), zap.String("customer_id", *existingCustomerID))
	} else if order.CustomerPhone != "" {
		// Try to find existing customer by phone in Shopify
		byPhone, err := s.FindCustomerIDByPhone(ctx, order.CustomerPhone)
		if err != nil {
			s.logger.Warn("Lookup customer by phone failed", zap.String("phone", order.CustomerPhone), zap.Error(err))
		}
		if byPhone != nil {
			customerIDToUse = byPhone
			s.logger.Info("Attaching draft order to Shopify customer found by phone", zap.String("phone", order.CustomerPhone), zap.String("customer_id", *byPhone))
		}
	}

	// Generated email from phone: used only when we don't have a customer ID (so Shopify creates/find by this email; we never use request email so matching is by phone only)
	var customerEmail *string
	if customerIDToUse == nil && order.CustomerPhone != "" {
		phoneDigits := strings.ReplaceAll(order.CustomerPhone, " ", "")
		phoneDigits = strings.ReplaceAll(phoneDigits, "-", "")
		phoneDigits = strings.ReplaceAll(phoneDigits, "(", "")
		phoneDigits = strings.ReplaceAll(phoneDigits, ")", "")
		if len(phoneDigits) > 10 {
			phoneDigits = phoneDigits[len(phoneDigits)-10:]
		}
		if phoneDigits != "" {
			email := fmt.Sprintf("phone-%s@example.com", phoneDigits)
			customerEmail = &email
			s.logger.Info("Using generated email from phone for new Shopify customer", zap.String("email", email), zap.String("phone", order.CustomerPhone))
		}
	}
	if customerIDToUse == nil && customerEmail == nil {
		if emailVal, ok := order.ShippingAddress["email"].(string); ok && emailVal != "" {
			customerEmail = &emailVal
			s.logger.Info("Using provided email (no phone)", zap.String("email", emailVal))
		}
	}

	// For existing customers, ensure Shopify customer address matches order shipping address; update if changed.
	if customerIDToUse != nil {
		if err := s.EnsureCustomerAddress(ctx, *customerIDToUse, order); err != nil {
			s.logger.Warn("Ensure customer address failed (draft order will still use order address)", zap.String("customer_id", *customerIDToUse), zap.Error(err))
		}
	}

	// Build input
	input := shopify.DraftOrderInput{
		LineItems:       lineItems,
		ShippingAddress: &shippingAddr,
		Tags:            tags,
		Note:            stringPtr(fmt.Sprintf("Partner Order ID: %s", order.PartnerOrderID)),
		Metafields: []shopify.MetafieldInput{
			{Namespace: "custom", Key: "parnters", Type: "single_line_text_field", Value: partnerName},
		},
	}

	if customerIDToUse != nil {
		input.CustomerID = customerIDToUse
	} else if customerEmail != nil {
		input.Email = customerEmail
	}

	// Execute mutation
	variables := map[string]interface{}{
		"input": input,
	}

	resp, err := s.client.Execute(shopify.DraftOrderCreateMutation, variables)
	if err != nil {
		return 0, fmt.Errorf("failed to create draft order: %w", err)
	}

	// Parse response to get draft order ID
	// NOTE: shopify.Client.Execute returns GraphQLResponse where resp.Data is already the "data" object.
	// So resp.Data looks like: { "draftOrderCreate": { ... } } (no outer {"data": ...} wrapper).
	var result struct {
		DraftOrderCreate struct {
			DraftOrder struct {
				ID string `json:"id"`
			} `json:"draftOrder"`
			UserErrors []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"draftOrderCreate"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return 0, fmt.Errorf("failed to parse draft order response: %w", err)
	}

	if len(result.DraftOrderCreate.UserErrors) > 0 {
		return 0, fmt.Errorf("shopify user errors: %v", result.DraftOrderCreate.UserErrors)
	}

	// Extract numeric ID from GID
	draftOrderGID := result.DraftOrderCreate.DraftOrder.ID
	draftOrderID, err := extractIDFromGID(draftOrderGID)
	if err != nil {
		return 0, fmt.Errorf("failed to extract draft order ID: %w", err)
	}

	return draftOrderID, nil
}

// Helper functions
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func stringPtr(s string) *string {
	return &s
}

func extractIDFromGID(gid string) (int64, error) {
	// GID format: "gid://shopify/DraftOrder/123456"
	parts := strings.Split(gid, "/")
	if len(parts) < 4 {
		return 0, fmt.Errorf("invalid GID format: %s", gid)
	}

	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ID from GID: %w", err)
	}

	return id, nil
}

// normalizePhoneForComparison strips non-digits and takes the last 10 digits so "0778888888", "+962 77 888 8888", "00962778888888" match.
func normalizePhoneForComparison(phone string) string {
	var digits strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	s := digits.String()
	if len(s) > 10 {
		return s[len(s)-10:]
	}
	return s
}
