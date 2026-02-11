package shopify

// DraftOrderCreateMutation creates a draft order
const DraftOrderCreateMutation = `
mutation draftOrderCreate($input: DraftOrderInput!) {
  draftOrderCreate(input: $input) {
    draftOrder {
      id
      name
      order {
        id
      }
    }
    userErrors {
      field
      message
    }
  }
}
`

// DraftOrderCompleteMutation completes a draft order and converts it into an order.
const DraftOrderCompleteMutation = `
mutation draftOrderComplete($id: ID!) {
  draftOrderComplete(id: $id) {
    draftOrder {
      id
      order {
        id
      }
    }
    userErrors {
      field
      message
    }
  }
}
`

// MetafieldsSetMutation sets metafields on a resource (e.g. Order). Used to set custom.parnters on the Order after draft complete.
const MetafieldsSetMutation = `
mutation metafieldsSet($metafields: [MetafieldsSetInput!]!) {
  metafieldsSet(metafields: $metafields) {
    metafields {
      key
      namespace
      value
    }
    userErrors {
      field
      message
      code
    }
  }
}
`

// DraftOrderInput represents the input for creating a draft order
type DraftOrderInput struct {
	LineItems       []DraftOrderLineItemInput `json:"lineItems"`
	CustomerID      *string                  `json:"customerId,omitempty"`
	Email           *string                  `json:"email,omitempty"`
	ShippingAddress *DraftOrderAddressInput  `json:"shippingAddress,omitempty"`
	Tags            []string                 `json:"tags,omitempty"`
	Note            *string                  `json:"note,omitempty"`
	CustomAttributes []DraftOrderAttributeInput `json:"customAttributes,omitempty"`
	Metafields      []MetafieldInput         `json:"metafields,omitempty"`
}

// MetafieldInput is used to set a metafield on a draft order (e.g. custom.partners)
type MetafieldInput struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

type DraftOrderLineItemInput struct {
	VariantID    *string  `json:"variantId,omitempty"`
	Title        *string  `json:"title,omitempty"`
	// For custom line items (no variantId), Shopify expects originalUnitPrice, not price.
	OriginalUnitPrice *string `json:"originalUnitPrice,omitempty"`
	Quantity     int      `json:"quantity"`
	CustomAttributes []DraftOrderAttributeInput `json:"customAttributes,omitempty"`
}

type DraftOrderAddressInput struct {
	FirstName    string  `json:"firstName"`
	LastName     *string `json:"lastName,omitempty"`
	Address1     string  `json:"address1"`
	Address2     *string `json:"address2,omitempty"`
	City         string  `json:"city"`
	Province     *string `json:"province,omitempty"`
	Zip          string  `json:"zip"`
	Country      string  `json:"country"`
	Phone        *string `json:"phone,omitempty"`
}

type DraftOrderAttributeInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// MetafieldsSetInput is used with metafieldsSet mutation (e.g. to set metafield on Order).
type MetafieldsSetInput struct {
	OwnerID   string `json:"ownerId"`
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

// CustomerAddressUpdateMutation updates a customer's address (Admin API 2025-04+).
const CustomerAddressUpdateMutation = `
mutation customerAddressUpdate($customerId: ID!, $addressId: ID!, $address: MailingAddressInput!, $setAsDefault: Boolean) {
  customerAddressUpdate(customerId: $customerId, addressId: $addressId, address: $address, setAsDefault: $setAsDefault) {
    address {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

// CustomerAddressCreateMutation creates a new address for a customer (Admin API 2025-04+).
const CustomerAddressCreateMutation = `
mutation customerAddressCreate($customerId: ID!, $address: MailingAddressInput!, $setAsDefault: Boolean) {
  customerAddressCreate(customerId: $customerId, address: $address, setAsDefault: $setAsDefault) {
    address {
      id
    }
    userErrors {
      field
      message
    }
  }
}
`

// MailingAddressInput is used for customerAddressUpdate and customerAddressCreate (countryCode: ISO 3166-1 alpha-2).
type MailingAddressInput struct {
	Address1     string  `json:"address1"`
	Address2     *string `json:"address2,omitempty"`
	City         string  `json:"city"`
	Company      *string `json:"company,omitempty"`
	CountryCode  string  `json:"countryCode"`
	FirstName    string  `json:"firstName"`
	LastName     string  `json:"lastName"`
	Phone        *string `json:"phone,omitempty"`
	ProvinceCode *string `json:"provinceCode,omitempty"`
	Zip          string  `json:"zip"`
}
