# B2B API Documentation

## Base URL

```
https://api.jafarshop.com/v1
```

## Authentication

All partner endpoints require API key authentication using the `Authorization` header:

```
Authorization: Bearer {api_key}
```

API keys are provided by JafarShop when setting up a partner account.

## Endpoints

### 1. Submit Cart

Submit a cart for order processing. The system will check if the cart contains any JafarShop products. If yes, a draft order will be created in Shopify.

**Endpoint:** `POST /v1/carts/submit`

**Headers:**

- `Authorization: Bearer {api_key}` (required)
- `Idempotency-Key: {uuid}` (optional, recommended)

**Request Body:**

```json
{
  "partner_order_id": "ORDER-2024-001",
  "items": [
    {
      "sku": "JS-PROD-001",
      "title": "JafarShop Product",
      "price": 29.99,
      "quantity": 2,
      "product_url": "https://partner-store.com/product/js-prod-001"
    },
    {
      "sku": "OTHER-001",
      "title": "Other Product",
      "price": 19.99,
      "quantity": 1,
      "product_url": "https://partner-store.com/product/other-001"
    }
  ],
  "customer": {
    "name": "John Doe",
    "phone": "+1234567890"
  },
  "shipping": {
    "street": "123 Main Street",
    "city": "New York",
    "state": "NY",
    "postal_code": "10001",
    "country": "US"
  },
  "totals": {
    "subtotal": 79.97,
    "tax": 6.40,
    "shipping": 5.00,
    "total": 91.37
  },
  "payment_status": "paid",
  "payment_method": "Credit Card"
}
```

**Response (200 OK):**

```json
{
  "supplier_order_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING_CONFIRMATION"
}
```

**Response (204 No Content):**

- Cart does not contain any JafarShop products
- No response body

**Response (409 Conflict):**

```json
{
  "error": "idempotency key conflict: same key used with different payload"
}
```

**Response (422 Unprocessable Entity):**

```json
{
  "error": "validation failed",
  "details": "field validation errors..."
}
```

### 2. Get Order Status

Retrieve the current status and details of an order.

**Endpoint:** `GET /v1/orders/{supplier_order_id}`

**Headers:**

- `Authorization: Bearer {api_key}` (required)

**Path Parameters:**

- `supplier_order_id` - UUID of the supplier order

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "partner_order_id": "ORDER-2024-001",
  "status": "CONFIRMED",
  "shopify_draft_order_id": 123456789,
  "customer_name": "John Doe",
  "customer_phone": "+1234567890",
  "shipping_address": {
    "street": "123 Main Street",
    "city": "New York",
    "state": "NY",
    "postal_code": "10001",
    "country": "US"
  },
  "cart_total": 91.37,
  "payment_status": "paid",
  "payment_method": "Credit Card",
  "items": [
    {
      "sku": "JS-PROD-001",
      "title": "JafarShop Product",
      "price": 29.99,
      "quantity": 2,
      "product_url": "https://partner-store.com/product/js-prod-001",
      "is_supplier_item": true,
      "shopify_variant_id": 987654321
    },
    {
      "sku": "OTHER-001",
      "title": "Other Product",
      "price": 19.99,
      "quantity": 1,
      "product_url": "https://partner-store.com/product/other-001",
      "is_supplier_item": false
    }
  ],
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:05:00Z"
}
```

**Response (404 Not Found):**

```json
{
  "error": "order not found"
}
```

**Response (403 Forbidden):**

```json
{
  "error": "access denied"
}
```

(Returned when the order belongs to a different partner)

### 3. Confirm Order (Admin)

Confirm an order for fulfillment.

**Endpoint:** `POST /v1/admin/orders/{id}/confirm`

**Headers:**

- `Authorization: Bearer {api_key}` (required)

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "CONFIRMED"
}
```

### 4. Reject Order (Admin)

Reject an order with a reason.

**Endpoint:** `POST /v1/admin/orders/{id}/reject`

**Headers:**

- `Authorization: Bearer {api_key}` (required)

**Request Body:**

```json
{
  "reason": "Product out of stock"
}
```

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "REJECTED"
}
```

### 5. Ship Order (Admin)

Mark an order as shipped with tracking information.

**Endpoint:** `POST /v1/admin/orders/{id}/ship`

**Headers:**

- `Authorization: Bearer {api_key}` (required)

**Request Body:**

```json
{
  "carrier": "Standard Shipping",
  "tracking_number": "TRACK123456789",
  "tracking_url": "https://example.com/track/TRACK123456789"
}
```

**Response (200 OK):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "SHIPPED",
  "tracking_carrier": "Standard Shipping",
  "tracking_number": "TRACK123456789",
  "tracking_url": "https://example.com/track/TRACK123456789"
}
```

### 6. List Orders (Admin)

List orders with optional filtering.

**Endpoint:** `GET /v1/admin/orders`

**Headers:**

- `Authorization: Bearer {api_key}` (required)

**Query Parameters:**

- `status` (optional) - Filter by status (PENDING_CONFIRMATION, CONFIRMED, REJECTED, SHIPPED, DELIVERED, CANCELLED)
- `limit` (optional, default: 50) - Number of results (1-100)
- `offset` (optional, default: 0) - Pagination offset

**Response (200 OK):**

```json
{
  "orders": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "partner_order_id": "ORDER-2024-001",
      "status": "CONFIRMED",
      "shopify_draft_order_id": 123456789,
      "customer_name": "John Doe",
      "cart_total": 91.37,
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:05:00Z"
    }
  ],
  "limit": 50,
  "offset": 0
}
```

## Order Statuses

- `PENDING_CONFIRMATION` - Order received, awaiting manual confirmation
- `CONFIRMED` - Order confirmed and ready for fulfillment
- `REJECTED` - Order rejected (with reason)
- `SHIPPED` - Order shipped (with tracking)
- `DELIVERED` - Order delivered (optional)
- `CANCELLED` - Order cancelled

## Payment Methods

Supported payment methods:
- `Card On Delivery` - Card payment on delivery
- `Cash On Delivery (COD)` - Cash payment on delivery
- `Credit Card` - Credit card payment
- `ZainCash` - ZainCash payment

## Idempotency

To prevent duplicate orders from retries, include an `Idempotency-Key` header with a unique value (UUID recommended) for each cart submission.

- Same key + same payload → Returns existing order
- Same key + different payload → Returns 409 Conflict
- Different key → Creates new order

Idempotency keys are valid for 24 hours.

## Error Handling

All errors follow this format:

```json
{
  "error": "Error message",
  "details": "Additional details (if applicable)"
}
```

### HTTP Status Codes

- `200 OK` - Success
- `204 No Content` - No supplier products in cart
- `400 Bad Request` - Invalid request
- `401 Unauthorized` - Invalid or missing API key
- `403 Forbidden` - Access denied
- `404 Not Found` - Resource not found
- `409 Conflict` - Idempotency conflict
- `422 Unprocessable Entity` - Validation error
- `500 Internal Server Error` - Server error

## Rate Limiting

Currently no rate limiting is implemented. This will be added in a future version.

## Support

For API support, contact: Feras.jafarShop@gmail.com
