# B2B API – All Endpoints Reference

This document lists **all endpoints** for **OrderB2bAPI** (orders & catalog for partners) and **ProductB2B** (Shopify catalog, menus, webhooks), what they do, and how to use them.

---

## Overview

| System        | Default URL              | Role |
|---------------|--------------------------|------|
| **OrderB2bAPI** | `http://localhost:8081` | Partner-facing API: catalog, cart submit, orders. Partners use their **partner API key**. |
| **ProductB2B**  | `http://localhost:3000` | Shopify catalog & menus. OrderB2bAPI calls it with **PRODUCT_B2B_SERVICE_API_KEY**; you can also call it directly for debug. |

---

## Authentication

### OrderB2bAPI (partner endpoints)

All `/v1/*` routes (except health) require:

```http
Authorization: Bearer <partner_api_key>
```

The partner API key is created when you run `create-partner` and is stored hashed; use the same key you received at creation.

### ProductB2B

- **Service key** (for OrderB2bAPI server-to-server and for `collection_handle`):
  ```http
  Authorization: Bearer <PRODUCT_B2B_SERVICE_API_KEY>
  ```
- **Partner keys** (from `PARTNER_API_KEYS` in ProductB2B .env) can also call catalog with the default collection.
- Some routes (e.g. `/health`, `/debug/sku-lookup`) require **no auth**.

---

# PostgreSQL Database Structure (OrderB2bAPI)

The **OrderB2bAPI** uses a PostgreSQL database. **ProductB2B** does not use this database; it talks to Shopify only. This section describes the schema: tables, primary keys, foreign keys, indexes, and important features.

---

## Extensions

| Extension   | Purpose |
|------------|---------|
| `uuid-ossp` | Generates UUID primary keys (`uuid_generate_v4()`). |

---

## Tables Overview

| Table                 | Primary key | Purpose |
|-----------------------|-------------|---------|
| `partners`            | `id` (UUID) | Partner accounts; API key hash, collection handle. |
| `sku_mappings`        | `id` (UUID) | Legacy global SKU → Shopify variant (optional). |
| `supplier_orders`     | `id` (UUID) | Orders submitted by partners (cart submit). |
| `supplier_order_items`| `id` (UUID) | Line items per order. |
| `idempotency_keys`    | `key` (VARCHAR) | Idempotency for cart submit. |
| `order_events`        | `id` (UUID) | Audit trail for order events. |
| `partner_sku_mappings`| `id` (UUID) | Per-partner catalog (SKU → Shopify, synced from ProductB2B). |

---

## Table Details

### 1. `partners`

Stores partner accounts. Each partner has an API key (stored as bcrypt hash) and an optional Shopify collection handle for their catalog.

| Column             | Type         | Nullable | Default | Description |
|--------------------|--------------|----------|---------|-------------|
| **id**             | UUID         | No       | `uuid_generate_v4()` | **Primary key.** |
| name               | VARCHAR(255) | No       | -       | Partner display name. |
| api_key_hash       | VARCHAR(255) | No       | -       | Bcrypt hash of API key. **UNIQUE.** |
| api_key_lookup     | VARCHAR(64)  | Yes      | -       | SHA256(api_key) hex for fast lookup. **UNIQUE** when set. |
| webhook_url        | VARCHAR(500) | Yes      | -       | Optional webhook URL. |
| collection_handle  | VARCHAR(255) | Yes      | -       | Shopify collection handle for this partner’s catalog. |
| is_active          | BOOLEAN      | No       | `true`  | If false, partner cannot authenticate. |
| created_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | |
| updated_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | Auto-updated by trigger. |

**Indexes:** `api_key_hash`, `is_active`, `api_key_lookup` (partial: where not null).

**Important:** Auth looks up by `api_key_lookup` (when set) then verifies with `api_key_hash` (bcrypt). New partners get `api_key_lookup` set at creation.

---

### 2. `sku_mappings` (legacy)

Global SKU → Shopify product/variant. Optional; cart validation can use `partner_sku_mappings` instead.

| Column             | Type         | Nullable | Default | Description |
|--------------------|--------------|----------|---------|-------------|
| **id**             | UUID         | No       | `uuid_generate_v4()` | **Primary key.** |
| sku                | VARCHAR(255) | No       | -       | **UNIQUE.** |
| shopify_product_id | BIGINT       | No       | -       | |
| shopify_variant_id | BIGINT       | No       | -       | |
| is_active          | BOOLEAN      | No       | `true`  | |
| created_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | |
| updated_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | Trigger. |

**Indexes:** `sku`, `is_active`.

---

### 3. `supplier_orders`

One row per order (cart submit). Partner reference and partner_order_id are unique per partner.

| Column               | Type         | Nullable | Default | Description |
|----------------------|--------------|----------|---------|-------------|
| **id**               | UUID         | No       | `uuid_generate_v4()` | **Primary key.** |
| partner_id           | UUID         | No       | -       | **FK → partners(id) ON DELETE RESTRICT.** |
| partner_order_id     | VARCHAR(255) | No       | -       | Partner’s own order id. **UNIQUE(partner_id, partner_order_id).** |
| status               | VARCHAR(50)  | No       | `'INCOMPLETE_CAUTION'` | Shopify-aligned (e.g. INCOMPLETE_CAUTION, UNFULFILLED, FULFILLED, COMPLETE, REJECTED, CANCELED). |
| shopify_draft_order_id | BIGINT     | Yes      | -       | Draft order in Shopify. |
| shopify_order_id     | BIGINT       | Yes      | -       | Finalized Shopify order id. |
| customer_name        | VARCHAR(255) | No       | -       | |
| customer_phone       | VARCHAR(50)  | Yes      | -       | |
| shipping_address     | JSONB        | No       | -       | Address object (city, address, etc.). |
| cart_total           | DECIMAL(10,2)| No       | -       | |
| payment_status       | VARCHAR(50)  | Yes      | -       | |
| payment_method       | VARCHAR(50)  | Yes      | -       | |
| rejection_reason     | VARCHAR(500) | Yes      | -       | |
| tracking_carrier     | VARCHAR(100) | Yes      | -       | |
| tracking_number      | VARCHAR(255) | Yes      | -       | |
| tracking_url         | VARCHAR(500) | Yes      | -       | |
| created_at           | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | |
| updated_at           | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | Trigger. |

**Indexes:** `partner_id`, `status`, `partner_order_id`, `shopify_draft_order_id`, `shopify_order_id`, `payment_method`.

**Important:** Deleting a partner is **RESTRICT**ed if they have orders; delete or reassign orders first.

---

### 4. `supplier_order_items`

Line items for each order.

| Column             | Type         | Nullable | Default | Description |
|--------------------|--------------|----------|---------|-------------|
| **id**             | UUID         | No       | `uuid_generate_v4()` | **Primary key.** |
| supplier_order_id  | UUID         | No       | -       | **FK → supplier_orders(id) ON DELETE CASCADE.** |
| sku                | VARCHAR(255) | No       | -       | |
| title              | VARCHAR(500) | No       | -       | |
| price              | DECIMAL(10,2)| No       | -       | |
| quantity           | INTEGER      | No       | -       | |
| product_url        | VARCHAR(500) | Yes      | -       | |
| is_supplier_item   | BOOLEAN      | No       | `false` | True if from supplier catalog. |
| shopify_variant_id | BIGINT       | Yes      | -       | |
| created_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | |

**Indexes:** `supplier_order_id`, `sku`, `is_supplier_item`.

**Important:** Deleting an order **CASCADE**s to its items.

---

### 5. `idempotency_keys`

Ensures the same Idempotency-Key + same body returns the same order (no duplicate orders).

| Column             | Type         | Nullable | Default | Description |
|--------------------|--------------|----------|---------|-------------|
| **key**            | VARCHAR(255) | No       | -       | **Primary key.** Idempotency-Key header value. |
| partner_id         | UUID         | No       | -       | **FK → partners(id) ON DELETE CASCADE.** |
| supplier_order_id  | UUID         | No       | -       | **FK → supplier_orders(id) ON DELETE CASCADE.** |
| request_hash       | VARCHAR(64)  | No       | -       | SHA256 of request body; same key + same hash = same order. |
| created_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | |

**Indexes:** `partner_id`, `supplier_order_id`.

**Important:** Same key with different body returns 409 Conflict. Deleting partner or order removes related idempotency rows.

---

### 6. `order_events`

Audit log for order lifecycle (event_type, event_data).

| Column             | Type         | Nullable | Default | Description |
|--------------------|--------------|----------|---------|-------------|
| **id**             | UUID         | No       | `uuid_generate_v4()` | **Primary key.** |
| supplier_order_id  | UUID         | No       | -       | **FK → supplier_orders(id) ON DELETE CASCADE.** |
| event_type         | VARCHAR(100) | No       | -       | |
| event_data         | JSONB        | Yes      | -       | |
| created_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | |

**Indexes:** `supplier_order_id`, `event_type`, `created_at`.

---

### 7. `partner_sku_mappings`

Per-partner product catalog: one row per (partner, SKU). Populated by catalog sync from ProductB2B; used for cart validation and order enrichment (title, image).

| Column             | Type         | Nullable | Default | Description |
|--------------------|--------------|----------|---------|-------------|
| **id**             | UUID         | No       | `uuid_generate_v4()` | **Primary key.** |
| partner_id         | UUID         | No       | -       | **FK → partners(id) ON DELETE CASCADE.** |
| sku                | VARCHAR(255) | No       | -       | **UNIQUE(partner_id, sku).** |
| shopify_product_id | BIGINT       | No       | -       | |
| shopify_variant_id | BIGINT       | No       | -       | |
| title              | VARCHAR(500) | Yes      | -       | Product title. |
| price              | VARCHAR(50)  | Yes      | -       | Display price. |
| image_url          | VARCHAR(500) | Yes      | -       | |
| is_active          | BOOLEAN      | No       | `true`  | |
| created_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | |
| updated_at         | TIMESTAMP    | No       | `CURRENT_TIMESTAMP` | Trigger. |

**Indexes:** `partner_id`, `(partner_id, sku)`, `is_active`.

**Important:** Catalog sync (on startup and every 10 min) upserts from ProductB2B per partner’s `collection_handle`. Used when ProductB2B is down (catalog fallback) and for order item enrichment.

---

## Relationships (ER summary)

```
partners (1) ──< supplier_orders     (partner_id, ON DELETE RESTRICT)
partners (1) ──< idempotency_keys    (partner_id, ON DELETE CASCADE)
partners (1) ──< partner_sku_mappings (partner_id, ON DELETE CASCADE)

supplier_orders (1) ──< supplier_order_items (supplier_order_id, ON DELETE CASCADE)
supplier_orders (1) ──< idempotency_keys    (supplier_order_id, ON DELETE CASCADE)
supplier_orders (1) ──< order_events         (supplier_order_id, ON DELETE CASCADE)
```

---

## Important Features

| Feature | Where | Purpose |
|--------|--------|---------|
| **UUID primary keys** | All main tables (except idempotency_keys) | Stable, non-guessable IDs. |
| **JSONB** | `supplier_orders.shipping_address`, `order_events.event_data` | Flexible structure; indexable if needed. |
| **Unique (partner_id, partner_order_id)** | `supplier_orders` | One order per partner per partner reference. |
| **Unique (partner_id, sku)** | `partner_sku_mappings` | One mapping per SKU per partner. |
| **api_key_lookup** | `partners` | SHA256(api_key) for fast auth lookup; bcrypt in `api_key_hash` for verification. |
| **Triggers** | `partners`, `sku_mappings`, `supplier_orders`, `partner_sku_mappings` | Auto-set `updated_at` on UPDATE. |
| **ON DELETE RESTRICT** | `supplier_orders.partner_id` | Prevent deleting a partner who has orders. |
| **ON DELETE CASCADE** | Order items, idempotency, events, partner_sku_mappings | Deleting order or partner cleans related rows. |

---

## Migrations (order applied)

| Migration | Description |
|-----------|-------------|
| 000001 | Init: partners, sku_mappings, supplier_orders, supplier_order_items, idempotency_keys, order_events; UUID extension; triggers. |
| 000002 | Add `payment_method` to supplier_orders. |
| 000003 | Add `shopify_order_id` to supplier_orders. |
| 000004 | Align order status values with Shopify (e.g. INCOMPLETE_CAUTION, UNFULFILLED, FULFILLED). |
| 000005 | Add `collection_handle` to partners. |
| 000006 | Add `partner_sku_mappings` table and indexes. |
| 000007 | Add `api_key_lookup` to partners (SHA256 hex for fast auth). |

---

# OrderB2bAPI Endpoints

Base URL: `http://localhost:8081` (or your deployed URL).

---

## 1. Health check

**What it does:** Returns service liveness. No authentication.

| Method | Path     | Auth |
|--------|----------|------|
| GET    | `/health` | No   |

**Response (200):**

```json
{ "status": "ok" }
```

**How to use:**

```bash
curl -s "http://localhost:8081/health"
```

---

## 2. Get catalog products (partner’s catalog)

**What it does:** Returns the **partner’s product catalog** (paginated). Data comes from ProductB2B using the partner’s `collection_handle`; if ProductB2B is unavailable, it falls back to the last synced data in `partner_sku_mappings`. Each partner only sees products for their assigned collection.

| Method | Path                   | Auth   |
|--------|------------------------|--------|
| GET    | `/v1/catalog/products` | Partner |

**Query parameters:**

| Parameter | Type   | Default | Description                |
|-----------|--------|---------|----------------------------|
| `limit`  | number | 25      | Page size (1–100).         |
| `cursor` | string | -       | Pagination cursor from previous response. |

**Response (200):**

```json
{
  "data": [
    {
      "id": "gid://shopify/Product/123",
      "title": "Product Name",
      "handle": "product-handle",
      "variants": { "nodes": [ { "sku": "SKU-001", "price": "10.00", ... } ] },
      "featuredImage": { "url": "..." },
      ...
    }
  ],
  "pagination": {
    "hasNextPage": false,
    "nextCursor": ""
  },
  "meta": {
    "collection": "partner-zain",
    "count": 5
  }
}
```

**How to use:**

```bash
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "http://localhost:8081/v1/catalog/products?limit=5"
```

---

## 3. Submit cart (create order)

**What it does:** Submits a cart and creates an order. Items are validated against the **partner’s catalog** (SKUs must exist in that partner’s products). If the cart contains valid supplier items, a Shopify draft order can be created. **Idempotency-Key** is recommended so duplicate requests return the same order.

| Method | Path               | Auth   |
|--------|--------------------|--------|
| POST   | `/v1/carts/submit` | Partner |

**Headers:**

| Header             | Required | Description |
|--------------------|----------|-------------|
| `Authorization`    | Yes      | `Bearer <partner_api_key>` |
| `Content-Type`     | Yes      | `application/json` |
| `Idempotency-Key`  | Recommended | Unique string per logical request (e.g. UUID or `order-{partner_order_id}`). Same key + same body returns existing order. |

**Request body:**

```json
{
  "partner_order_id": "YOUR-ORDER-ID-123",
  "items": [
    {
      "sku": "SKU-FROM-CATALOG",
      "title": "Product Title",
      "price": 10.00,
      "quantity": 1,
      "product_url": "https://optional.com/product"
    }
  ],
  "customer": {
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "phone_number": "+962700000000"
  },
  "shipping": {
    "city": "Amman",
    "area": "Downtown",
    "address": "123 Main St",
    "postal_code": "11181",
    "country": "Jordan"
  },
  "totals": {
    "subtotal": 10.00,
    "tax": 0,
    "shipping": 0,
    "total": 10.00
  }
}
```

- **partner_order_id:** Your unique order reference.
- **items:** At least one item; `sku` must exist in the partner’s catalog.
- **customer:** `first_name`, `last_name`, `phone_number` required; `email` optional.
- **shipping:** `city`, `address` required; `area`, `postal_code`, `country` optional.
- **totals:** Must be consistent with items (e.g. total ≥ subtotal).

**Response (200):**

```json
{
  "supplier_order_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING_CONFIRMATION",
  "shopify_draft_order_id": 1234567890,
  "shopify_order_id": null,
  "shopify_error": ""
}
```

**Other responses:**

- **400** – Validation error (e.g. invalid body).
- **409** – Idempotency conflict (same key, different body).
- **422** – Cart validation failed (e.g. SKU not in partner catalog).

**How to use:**

```bash
curl -s -X POST "http://localhost:8081/v1/carts/submit" \
  -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  -H "Idempotency-Key: unique-key-$(date +%s)" \
  -H "Content-Type: application/json" \
  -d '{"partner_order_id":"order-001","items":[{"sku":"REAL-SKU","title":"Product","price":10,"quantity":1}],"customer":{"first_name":"Test","last_name":"User","email":"t@e.com","phone_number":"+962700000000"},"shipping":{"city":"Amman","area":"","address":"123 St","postal_code":"","country":"Jordan"},"totals":{"subtotal":10,"tax":0,"shipping":0,"total":10}}'
```

---

## 4. Get order by ID or partner order ID

**What it does:** Returns a single order for the authenticated partner. You can pass either the **supplier order UUID** or your **partner_order_id**. Response includes items and, when available, product title and image from the partner catalog.

| Method | Path              | Auth   |
|--------|-------------------|--------|
| GET    | `/v1/orders/:id`  | Partner |

**Path:**

- `:id` – Either the supplier order UUID (e.g. `550e8400-e29b-41d4-a716-446655440000`) or your `partner_order_id` (e.g. `order-001`).

**Response (200):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "partner_order_id": "order-001",
  "status": "PENDING_CONFIRMATION",
  "shopify_draft_order_id": 1234567890,
  "shopify_order_id": null,
  "customer_name": "Test User",
  "customer_phone": "+962700000000",
  "shipping_address": { "city": "Amman", "address": "123 St", ... },
  "cart_total": 10.00,
  "payment_status": "",
  "items": [
    {
      "sku": "SKU-001",
      "title": "Product Title",
      "price": 10,
      "quantity": 1,
      "product_title": "Enriched Title",
      "product_image_url": "https://..."
    }
  ],
  "created_at": "2026-02-04T12:00:00Z",
  "updated_at": "2026-02-04T12:00:00Z"
}
```

**Other responses:**

- **404** – Order not found.
- **403** – Order belongs to another partner.

**How to use:**

```bash
# By supplier order UUID
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "http://localhost:8081/v1/orders/550e8400-e29b-41d4-a716-446655440000"

# By your partner_order_id
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "http://localhost:8081/v1/orders/order-001"
```

---

## 5. List orders (admin)

**What it does:** Lists orders for the authenticated partner with optional status filter and pagination. Used to see all orders and then call “Get order” for details.

| Method | Path                 | Auth   |
|--------|----------------------|--------|
| GET    | `/v1/admin/orders`   | Partner |

**Query parameters:**

| Parameter | Type   | Default | Description |
|-----------|--------|---------|-------------|
| `limit`  | number | 50      | Max results (1–100). |
| `offset` | number | 0       | Skip first N orders. |
| `status` | string | -       | Filter by status (e.g. `PENDING_CONFIRMATION`, `CONFIRMED`, `REJECTED`, `SHIPPED`). |

**Response (200):**

```json
{
  "orders": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "partner_order_id": "order-001",
      "status": "PENDING_CONFIRMATION",
      "shopify_draft_order_id": 1234567890,
      "customer_name": "Test User",
      "cart_total": 10.00,
      "created_at": "2026-02-04T12:00:00Z",
      "updated_at": "2026-02-04T12:00:00Z"
    }
  ],
  "limit": 50,
  "offset": 0
}
```

**How to use:**

```bash
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "http://localhost:8081/v1/admin/orders?limit=10"

# With status filter
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "http://localhost:8081/v1/admin/orders?status=PENDING_CONFIRMATION&limit=10"
```

---

# ProductB2B Endpoints

Base URL: `http://localhost:3000`. Used by OrderB2bAPI for catalog; you can also call it for debugging.

---

## 1. Health check

**What it does:** Liveness check. No auth.

| Method | Path     | Auth |
|--------|----------|------|
| GET    | `/health` | No   |

**Response:** Plain text `OK` or similar.

**How to use:**

```bash
curl -s "http://localhost:3000/health"
```

---

## 2. Catalog products (list or by SKU)

**What it does:** Returns products from a Shopify collection (by handle or default). When called with the **service key** and `collection_handle`, returns that collection’s products (used by OrderB2bAPI). With `sku`, returns a single product by SKU.

| Method | Path                   | Auth        |
|--------|------------------------|-------------|
| GET    | `/v1/catalog/products` | Service key or partner key |

**Query parameters:**

| Parameter            | Required | Description |
|-----------------------|----------|-------------|
| `collection_handle`   | When using service key | Shopify collection handle (e.g. `partner-zain`). |
| `sku`                 | For single product | Return one product by SKU. |
| `limit`               | No       | Page size (1–100), default 25. |
| `cursor`              | No       | Pagination cursor. |
| `lang`                | No       | `en` or `ar` for translations. |

**How to use:**

```bash
# List products for a collection (service key)
curl -s -H "Authorization: Bearer YOUR_PRODUCT_B2B_SERVICE_API_KEY" \
  "http://localhost:3000/v1/catalog/products?collection_handle=partner-zain&limit=5"

# Single product by SKU
curl -s -H "Authorization: Bearer YOUR_PRODUCT_B2B_SERVICE_API_KEY" \
  "http://localhost:3000/v1/catalog/products?sku=MK4820b"
```

---

## 3. Menus (all menus with nested items)

**What it does:** Returns Shopify Online Store menus with nested structure.

| Method | Path     | Auth |
|--------|----------|------|
| GET    | `/menus` | -    |

**How to use:**

```bash
curl -s "http://localhost:3000/menus"
```

---

## 4. Menu path by SKU

**What it does:** Returns the product name and menu hierarchy (path) for a given SKU.

| Method | Path                  | Auth |
|--------|-----------------------|------|
| GET    | `/menu-path-by-sku`    | -    |

**Query:** `sku` (required).

**How to use:**

```bash
curl -s "http://localhost:3000/menu-path-by-sku?sku=MK4820b"
```

---

## 5. Debug – SKU lookup (no auth)

**What it does:** Checks if a SKU exists and returns the Shopify product GID. No authentication; for testing only.

| Method | Path                 | Auth |
|--------|----------------------|------|
| GET    | `/debug/sku-lookup`  | No   |

**Query:** `sku` (required).

**Response (200):**

```json
{ "sku": "MK4820b", "productId": "gid://shopify/Product/123", "found": true }
```

**How to use:**

```bash
curl -s "http://localhost:3000/debug/sku-lookup?sku=MK4820b"
```

---

## 6. Debug – Partner products (default collection)

**What it does:** Returns products from the default partner collection (from `PARTNER_COLLECTION_HANDLE` / `PARTNER_COLLECTION_TITLE`). Useful to verify Shopify collection content.

| Method | Path                     | Auth |
|--------|--------------------------|------|
| GET    | `/debug/partner-products`| No   |

**How to use:**

```bash
curl -s "http://localhost:3000/debug/partner-products"
```

---

## 7. Debug – List collections

**What it does:** Lists Shopify collections (handle, title, etc.) so you can confirm collection handles for `collection_handle` queries.

| Method | Path                    | Auth |
|--------|-------------------------|------|
| GET    | `/debug/list-collections`| -    |

**How to use:**

```bash
curl -s "http://localhost:3000/debug/list-collections"
```

---

## 8. Debug – Translations

**What it does:** Returns translated fields for a product by GID and locale.

| Method | Path                    | Auth |
|--------|-------------------------|------|
| GET    | `/debug/translations`    | -    |

**Query:** `product_id` (Shopify GID), `locale` (e.g. `en`, `ar`).

**How to use:**

```bash
curl -s "http://localhost:3000/debug/translations?product_id=gid://shopify/Product/123&locale=en"
```

---

## 9. Webhooks (Shopify)

**What they do:** Receive Shopify webhooks for product/inventory changes. Called by Shopify, not by you directly in normal testing.

| Method | Path                                  | Auth / Caller   |
|--------|---------------------------------------|-----------------|
| POST   | `/webhooks/products/update`           | Shopify (HMAC)  |
| POST   | `/webhooks/products/delete`           | Shopify (HMAC)  |
| POST   | `/webhooks/inventory_levels/update`   | Shopify (HMAC)  |

---

## 10. OAuth (Shopify app install)

**What it does:** Start and complete Shopify OAuth for the ProductB2B app.

| Method | Path             | Auth |
|--------|------------------|------|
| GET    | `/auth`          | No   | Start: `?shop=YOURSTORE.myshopify.com` |
| GET    | `/auth/callback` | No   | Callback after merchant approves.     |

**How to use (browser or redirect):**

```
http://localhost:3000/auth?shop=YOURSTORE.myshopify.com
```

---

# Quick reference table

## OrderB2bAPI (port 8081)

| Method | Path                   | Auth   | Description |
|--------|------------------------|--------|-------------|
| GET    | `/health`              | No     | Health check |
| GET    | `/v1/catalog/products` | Partner | Partner’s catalog (paginated) |
| POST   | `/v1/carts/submit`     | Partner + Idempotency-Key | Submit cart / create order |
| GET    | `/v1/orders/:id`       | Partner | Get order by UUID or partner_order_id |
| GET    | `/v1/admin/orders`     | Partner | List partner’s orders |

## ProductB2B (port 3000)

| Method | Path                     | Auth        | Description |
|--------|--------------------------|-------------|-------------|
| GET    | `/health`                | No          | Health check |
| GET    | `/v1/catalog/products`   | Service/partner | Catalog by collection or SKU |
| GET    | `/menus`                 | -           | All menus (nested) |
| GET    | `/menu-path-by-sku`      | -           | Menu path for a SKU |
| GET    | `/debug/sku-lookup`      | No          | SKU → product GID |
| GET    | `/debug/partner-products`| No          | Default collection products |
| GET    | `/debug/list-collections`| -           | List Shopify collections |
| GET    | `/debug/translations`    | -           | Product translations by GID + locale |
| GET    | `/auth`                  | No          | Start Shopify OAuth |

---

*Replace `YOUR_PARTNER_API_KEY` and `YOUR_PRODUCT_B2B_SERVICE_API_KEY` with your real keys. When running in Docker, use the same host and ports (e.g. 8081 and 3000) or your deployed base URLs.*
