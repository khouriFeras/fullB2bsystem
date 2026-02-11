package service

// CartSubmitRequest represents the cart submission payload
type CartSubmitRequest struct {
	PartnerOrderID string                 `json:"partner_order_id" binding:"required"`
	Items          []CartItem             `json:"items" binding:"required,min=1"`
	Customer       CustomerInfo            `json:"customer" binding:"required"`
	Shipping       ShippingAddress         `json:"shipping" binding:"required"`
	Totals         CartTotals             `json:"totals" binding:"required"`
	PaymentMethod  *string                `json:"payment_method,omitempty"`
}

type CartItem struct {
	SKU        string  `json:"sku" binding:"required"`
	Title      string  `json:"title" binding:"required"`
	Price      float64 `json:"price" binding:"required,min=0"`
	Quantity   int     `json:"quantity" binding:"required,min=1"`
	ProductURL *string `json:"product_url,omitempty"`
}

// CustomerInfo matches the format received from Zain: first name, last name, email, phone.
type CustomerInfo struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email"`
	Phone     string `json:"phone_number" binding:"required"`
}

// ShippingAddress matches the format received from Zain: city, area, address. Country defaults to Jordan.
type ShippingAddress struct {
	City       string `json:"city" binding:"required"`
	Area       string `json:"area"`
	Address    string `json:"address" binding:"required"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type CartTotals struct {
	Subtotal float64 `json:"subtotal" binding:"required,min=0"`
	Tax      float64 `json:"tax" binding:"min=0"`
	Shipping float64 `json:"shipping" binding:"min=0"`
	Total    float64 `json:"total" binding:"required,min=0"`
}