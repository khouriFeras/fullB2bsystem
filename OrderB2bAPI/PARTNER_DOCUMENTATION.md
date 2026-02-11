# JafarShop Partner API — What You Need to Know

This guide explains everything we provide for order creation how to recive order information how to check the status of the orders and how to use it.

---

## What We Provide

As a partner you can:

1. **Submit orders** — Send us a cart when a customer checks out. We create an order for the items we supply.
2. **Check an order** — Get the current status and details of any order (using our order ID or your order ID).
3. **List your orders** — See all your orders, with optional filters and pagination.

You only see and manage your own orders. Other partners’ orders are not visible to you.

---

## Catalog and inventory

We maintain your catalog mapping and inventory in our database. When you submit a cart, we match the SKUs in your cart against your mapped catalog and check inventory. Only items that are in your catalog and in stock are fulfilled by us; other items are ignored for our side (you still handle them on your side). Your catalog is set up when you onboard, and inventory is kept up to date so orders are only created for products we can supply.

---

## Get Started

You need two things:

- **API base URL:** `https://api.jafarshop.com` (use `/v1` for all endpoints below)
- **API key** — We give you a secret key when your partner account is set up. Keep it private and never share it or put it in public code.

---

## Authentication

Every request must include your API key in the header:

```
Authorization: Bearer YOUR_API_KEY
```

Example:

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" https://api.jafarshop.com/v1/orders/...
```

If the key is missing or wrong, you will get `401 Unauthorized`.

---

## 1. Submit a Cart (Create an Order)

When a customer completes a purchase on your side, send us the cart. We check each item’s SKU against your catalog and our inventory database. If the cart contains any items we supply and have in stock, we create an order and return its ID. If the cart has no such items, we return success with no order (see below).

**Endpoint:** `POST /v1/carts/submit`

**Headers:**

| Header                                 | Required         | Description                                                           |
| -------------------------------------- | ---------------- | --------------------------------------------------------------------- |
| `Authorization: Bearer YOUR_API_KEY` | Yes              | Your API key                                                          |
| `Content-Type: application/json`     | Yes              | Request body is JSON                                                  |
| `Idempotency-Key: UNIQUE_VALUE`      | No (recommended) | A unique value per submission to avoid duplicate orders (e.g. a UUID) |

**Request body:**

| Field                | Required | Description                                         |
| -------------------- | -------- | --------------------------------------------------- |
| `partner_order_id` | Yes      | Your own order/reference ID (e.g. "ORDER-2024-001") |
| `items`            | Yes      | List of line items (see below)                      |
| `customer`         | Yes      | Customer name (and optional phone)                  |
| `shipping`         | Yes      | Full shipping address                               |
| `totals`           | Yes      | Subtotal, tax, shipping, total                      |
| `payment_status`   | No       | e.g. "paid"                                         |
| `payment_method`   | No       | e.g. "Credit Card", "Cash On Delivery (COD)"        |

**Each item in `items`:**

| Field           | Required | Description                                                           |
| --------------- | -------- | --------------------------------------------------------------------- |
| `sku`         | Yes      | Product SKU (must exist in your catalog and pass our inventory check) |
| `title`       | Yes      | Product title                                                         |
| `price`       | Yes      | Unit price (number ≥ 0)                                              |
| `quantity`    | Yes      | Quantity (integer ≥ 1)                                               |
| `product_url` | No       | Link to the product on your site                                      |

**Example request:**

```json
{
  "partner_order_id": "ORDER-2024-001",
  "items": [
    {
      "sku": "JS-PROD-001",
      "title": "Product Name",
      "price": 29.99,
      "quantity": 2,
      "product_url": "https://your-store.com/product/js-prod-001"
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
    "subtotal": 59.98,
    "tax": 4.80,
    "shipping": 5.00,
    "total": 69.78
  },
  "payment_status": "paid",
  "payment_method": "Credit Card"
}
```

**Responses:**

| Status                             | Meaning                                                                                                                                                                   |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **200 OK**                   | Cart had our products. We created (or already had) an order. Body:`{ "supplier_order_id": "uuid", "status": "..." }`. Save `supplier_order_id` to check status later. |
| **204 No Content**           | Cart had no products we supply. No order created. No response body. Treat as success.                                                                                     |
| **409 Conflict**             | You sent the same `Idempotency-Key` before with a different payload. Change the key or the payload.                                                                     |
| **422 Unprocessable Entity** | Validation failed (e.g. missing required field). Check the `error` and `details` in the body.                                                                         |

**Idempotency:** Sending the same `Idempotency-Key` and same body again returns the same order (200) instead of creating a duplicate. Use a new key for each new submission. Keys are valid for 24 hours.

---

## 2. Get Order Details and Status

Fetch one order by either our order ID (`supplier_order_id`) or your `partner_order_id`.

**Endpoint:** `GET /v1/orders/{id}`

**`{id}`** = either:

- The `supplier_order_id` we returned when you submitted the cart, or
- Your `partner_order_id` (e.g. `ORDER-2024-001`)

**Headers:** `Authorization: Bearer YOUR_API_KEY`

**Example:** `GET /v1/orders/550e8400-e29b-41d4-a716-446655440000`
or
`GET /v1/orders/ORDER-2024-001`

**Response (200 OK):**

You get the full order: status, customer, shipping, totals, and line items. For each item we tell you whether we supply it (`is_supplier_item`: true/false). When the order is shipped, we include tracking info when available.

Example (fields you care about):

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "partner_order_id": "ORDER-2024-001",
  "status": "UNFULFILLED",
  "customer_name": "feras khouri",
  "customer_phone": "+1234567890",
  "shipping_address": {
    "street": "mkka streat",
    "city": "amman",
    "state": "khladah",
    "postal_code": "00962",
    "country": "Jordan"
  },
  "cart_total": 69.78,
  "payment_status": "paid",
  "payment_method": "Credit Card",
  "items": [
    {
      "sku": "JS-PROD-001",
      "title": "Product Name",
      "price": 29.99,
      "quantity": 2,
      "product_url": "https://your-store.com/product/js-prod-001",
      "is_supplier_item": true
    }
  ],
  "tracking_carrier": "Standard Shipping",
  "tracking_number": "TRACK123456789",
  "tracking_url": "https://example.com/track/TRACK123456789",
  "rejection_reason": null,
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:05:00Z"
}
```

Tracking and rejection fields appear when relevant. You may see extra internal IDs in the response; you can ignore them.

**Other responses:**

| Status                  | Meaning                                      |
| ----------------------- | -------------------------------------------- |
| **404 Not Found** | No order with that ID for your account.      |
| **403 Forbidden** | Order exists but belongs to another partner. |

---

## 3. List Your Orders

Get a list of your orders, with optional filter by status and pagination.

**Endpoint:** `GET /v1/admin/orders`

**Headers:** `Authorization: Bearer YOUR_API_KEY`

**Query parameters:**

| Parameter  | Required | Description                                                  |
| ---------- | -------- | ------------------------------------------------------------ |
| `status` | No       | Filter by status (see list below). Omit to get all statuses. |
| `limit`  | No       | Number of orders per page (1–100, default 50).              |
| `offset` | No       | Number of orders to skip (default 0). Use for pagination.    |

**Example:** `GET /v1/admin/orders?status=UNFULFILLED&limit=20&offset=0`

**Response (200 OK):**

```json
{
  "orders": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "partner_order_id": "ORDER-2024-001",
      "status": "UNFULFILLED",
      "customer_name": "John Doe",
      "cart_total": 69.78,
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:05:00Z"
    }
  ],
  "limit": 50,
  "offset": 0
}
```

---

## Order Statuses

When you submit a cart or read an order, `status` will be one of these:

| Status                 | Meaning                                                        |
| ---------------------- | -------------------------------------------------------------- |
| `INCOMPLETE_CAUTION` | New order, pending our confirmation.                           |
| `UNFULFILLED`        | Confirmed, not yet shipped.                                    |
| `FULFILLED`          | Shipped (tracking may be in the order details).                |
| `COMPLETE`           | Delivered.                                                     |
| `REJECTED`           | We rejected the order (see `rejection_reason` on the order). |
| `CANCELED`           | Order canceled.                                                |
| `REFUNDED`           | Order refunded.                                                |
| `ARCHIVED`           | Archived.                                                      |

We may also return older status names (e.g. `PENDING_CONFIRMATION`, `CONFIRMED`, `SHIPPED`, `DELIVERED`, `CANCELLED`) for existing orders; treat them like the matching names above.

---

## Payment Methods

You can send any value in `payment_method`; we support and display common ones such as:

- Card On Delivery
- Cash On Delivery (COD)
- Credit Card
- ZainCash

---

## Errors

Failed responses use JSON like:

```json
{
  "error": "Short message",
  "details": "More detail when available"
}
```

**Common status codes:**

| Code | Meaning                                              |
| ---- | ---------------------------------------------------- |
| 200  | Success                                              |
| 204  | Success, no order created (cart had no our products) |
| 400  | Bad request (e.g. invalid parameter)                 |
| 401  | Invalid or missing API key                           |
| 403  | You don’t have access to this resource              |
| 404  | Resource not found                                   |
| 409  | Idempotency conflict (same key, different payload)   |
| 422  | Validation error (check `details`)                 |
| 500  | Server error; retry later or contact support         |

---

## Quick Reference

| What you want                              | Method | Endpoint             |
| ------------------------------------------ | ------ | -------------------- |
| Submit a cart / create order               | POST   | `/v1/carts/submit` |
| Get one order (by our ID or your order ID) | GET    | `/v1/orders/{id}`  |
| List your orders (with optional filters)   | GET    | `/v1/admin/orders` |

**Always send:** `Authorization: Bearer YOUR_API_KEY`
**For submit:** Prefer `Idempotency-Key: <unique-value>` to avoid duplicate orders.

---

## Support

- **Email:** feras.jafarShop@gmail.com
- For API key, catalog and inventory setup,integration help,anyhelp or clarification thats needed ,please reachout.
